package handlers

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/thetkpark/cscms-temp-storage/data"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"github.com/thetkpark/cscms-temp-storage/service/encrypt"
	"github.com/thetkpark/cscms-temp-storage/service/storage"
	"github.com/thetkpark/cscms-temp-storage/service/token"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
	"time"
)

type FileRoutesHandler struct {
	log               *zap.SugaredLogger
	encryptionManager encrypt.Manager
	fileDataStore     data.FileDataStore
	storageManager    storage.FileManager
	tokenManager      token.Manager
	maxStoreDuration  time.Duration
}

func NewFileRoutesHandler(log *zap.SugaredLogger, enc encrypt.Manager, data data.FileDataStore, store storage.FileManager, token token.Manager, duration time.Duration) *FileRoutesHandler {
	return &FileRoutesHandler{
		log:               log,
		encryptionManager: enc,
		fileDataStore:     data,
		storageManager:    store,
		tokenManager:      token,
		maxStoreDuration:  duration,
	}
}

// UploadFile handlers
// @Summary Upload new file
// @Description Upload new temporary store file
// @Tags File
// @Accept  multipart/form-data
// @Produce  json
// @Param       file  formData  file  true  "File"
// @Success      201  {object}  model.File
// @Failure      400  {object}  handlers.ErrorResponse
// @Failure      500  {object}  handlers.ErrorResponse
// @Router /api/file [post]
func (h *FileRoutesHandler) UploadFile(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to get file from form-data", err)
	}

	// Check image size (100MB)
	if fileHeader.Size > 100<<20 {
		return NewHTTPError(h.log, fiber.StatusRequestEntityTooLarge, "File too large", nil)
	}
	// Check slug
	t, err := h.tokenManager.GenerateFileToken()
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to generate file token", err)
	}
	fileToken := strings.ToLower(c.Query("slug", t))
	// Check if slug is available
	existingFile, err := h.fileDataStore.FindByToken(fileToken)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to get existing file token", err)
	}
	if existingFile != nil {
		return NewHTTPError(h.log, fiber.StatusBadRequest, fmt.Sprintf("%s slug is used", fileToken), nil)
	}

	// Check store duration (in day)
	storeDuration := h.maxStoreDuration
	if dayString := c.Query("duration"); len(dayString) > 0 {
		day, err := strconv.Atoi(dayString)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "duration must be integer")
		}
		if float64(day*24) > h.maxStoreDuration.Hours() {
			return NewHTTPError(h.log, fiber.StatusBadRequest, fmt.Sprintf("duration exceed maximum store duration (%v)", h.maxStoreDuration.Hours()/24), nil)
		}
		storeDuration = time.Duration(day) * time.Hour * 24
	}

	// Generate new file ID
	fileId, err := h.tokenManager.GenerateFileID()
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to create file id", err)
	}

	// Create new fileInfo struct
	fileInfo := &model.File{
		ID:        fileId,
		Token:     fileToken,
		Nonce:     "",
		Filename:  fileHeader.Filename,
		FileSize:  uint64(fileHeader.Size),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		ExpiredAt: time.Now().UTC().Add(storeDuration),
		Visited:   0,
		UserID:    0,
		FileType:  fileHeader.Header.Get("Content-Type"),
		Encrypted: false,
	}

	// Get userId if exist
	user := c.UserContext().Value("user")
	if user != nil {
		userModel, ok := user.(*model.User)
		if !ok {
			return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse to user model", fmt.Errorf("user model convertion error"))
		}
		fileInfo.UserID = userModel.ID
		fileInfo.Encrypted = true
	}

	// Open file from multipart form header
	var file io.Reader
	file, err = fileHeader.Open()
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to open file", err)
	}

	if fileInfo.Encrypted {
		// Encrypt the file
		file, fileInfo.Nonce, err = h.encryptionManager.Encrypt(file)
		if err != nil {
			return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable encrypt the file", err)
		}
	}

	// Write file content to disk
	if err := h.storageManager.WriteToNewFile(fileId, file); err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to write encrypted data to file", err)
	}

	err = h.fileDataStore.Create(fileInfo)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to save file info to db", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fileInfo)
}

// GetFile handlers
// @Summary Download the file
// @Description Access link to download the file
// @Tags File
// @Produce  application/octet-stream
// @Param        token       path      string      true  "File Token"
// @Success      200
// @Failure      500  {object}  handlers.ErrorResponse
// @Router /{token} [get]
func (h *FileRoutesHandler) GetFile(c *fiber.Ctx) error {
	t := strings.ToLower(c.Params("token"))

	// Find file by token
	fileInfo, err := h.fileDataStore.FindByToken(t)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to get file query", err)
	}
	if fileInfo == nil {
		return c.Redirect(c.BaseURL() + "/404")
	}

	// Check if file still exist on storage
	if exist, err := h.storageManager.Exist(fileInfo.ID); !exist {
		if err == nil {
			// File is not exist anymore
			return c.Redirect(c.BaseURL() + "/404")
		}
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to check if file exist", err)
	}

	// Get encrypted file from storage manager
	file, err := h.storageManager.OpenFile(fileInfo.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to open encrypted file", err)
	}

	if fileInfo.Encrypted {
		// Decrypt file if encrypted
		file, err = h.encryptionManager.Decrypt(file, fileInfo.Nonce)
		if err != nil {
			return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to decrypt", err)
		}
	}

	// Increase visited count
	err = h.fileDataStore.IncreaseVisited(fileInfo.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to increase count", err)
	}

	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileInfo.Filename))
	c.Set("Content-Type", "application/octet-stream")

	return c.SendStream(file, int(fileInfo.FileSize))
}

// GetOwnFiles handlers
// @Summary List of uploaded file
// @Description List all the upload file by the user
// @Tags File
// @Produce  json
// @Success      200  {array} model.File
// @Failure      500  {object}  handlers.ErrorResponse
// @Router /api/file [get]
func (h *FileRoutesHandler) GetOwnFiles(c *fiber.Ctx) error {
	// Get userId
	user := c.UserContext().Value("user")
	userModel, ok := user.(*model.User)
	if !ok {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse to user model", fmt.Errorf("user model convertion error"))
	}

	files, err := h.fileDataStore.FindByUserID(userModel.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "Unable to find files by user ID", err)
	}

	return c.JSON(files)
}

func (h *FileRoutesHandler) IsOwnFile(c *fiber.Ctx) error {
	fileId := c.Params("fileID", "")
	if len(fileId) == 0 {
		return NewHTTPError(h.log, fiber.StatusBadRequest, "File ID must be provided", nil)
	}

	// Get userId
	user := c.UserContext().Value("user")
	userModel, ok := user.(*model.User)
	if !ok {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse to user model", fmt.Errorf("user model convertion error"))
	}

	file, err := h.fileDataStore.FindByID(fileId)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to find file by id", err)
	}
	if file == nil || file.ExpiredAt.UTC().Before(time.Now().UTC()) {
		return NewHTTPError(h.log, fiber.StatusNotFound, "File not found", nil)
	}
	if file.UserID != userModel.ID {
		return NewHTTPError(h.log, fiber.StatusForbidden, "Forbidden", nil)
	}

	c.SetUserContext(context.WithValue(c.UserContext(), "file", file))
	return c.Next()
}

// DeleteFile handlers
// @Summary Delete the file
// @Description Delete the active file by ID
// @Tags File
// @Produce  json
// @Param        fileID       path      string      true  "File ID"
// @Success      200  {object} model.File
// @Failure      400  {object}  handlers.ErrorResponse
// @Failure      401  {object}  handlers.ErrorResponse
// @Failure      403  {object}  handlers.ErrorResponse
// @Failure      500  {object}  handlers.ErrorResponse
// @Router /api/file/{fileID} [delete]
func (h *FileRoutesHandler) DeleteFile(c *fiber.Ctx) error {
	fileModel, ok := c.UserContext().Value("file").(*model.File)
	if !ok {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse file model", fmt.Errorf("unable to parse file model"))
	}

	// Delete file record in db
	err := h.fileDataStore.DeleteByID(fileModel.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to delete file record in db", err)
	}

	// Delete file on storage
	err = h.storageManager.DeleteFile(fileModel.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to delete file on storage", err)
	}

	return c.JSON(fileModel)
}

// EditToken handlers
// @Summary Edit file token
// @Description Edit the file token/slug
// @Tags File
// @Produce  json
// @Param        fileID       path      string      true  "File ID"
// @Param        token       query      string      true  "New file token"
// @Success      200  {object} model.File
// @Failure      400  {object}  handlers.ErrorResponse
// @Failure      401  {object}  handlers.ErrorResponse
// @Failure      403  {object}  handlers.ErrorResponse
// @Failure      500  {object}  handlers.ErrorResponse
// @Router /api/file/{fileID} [patch]
func (h *FileRoutesHandler) EditToken(c *fiber.Ctx) error {
	fileModel, ok := c.UserContext().Value("file").(*model.File)
	if !ok {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse file model", fmt.Errorf("unable to parse file model"))
	}

	newToken := c.Query("token", "")
	if len(newToken) == 0 {
		return NewHTTPError(h.log, fiber.StatusBadRequest, "New Token must be provided", nil)
	}

	existingFile, err := h.fileDataStore.FindByToken(newToken)
	if existingFile != nil {
		return NewHTTPError(h.log, fiber.StatusBadRequest, "New token is in used", nil)
	}
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to query existing file with token", err)
	}

	fileModel.Token = newToken
	err = h.fileDataStore.UpdateToken(fileModel.ID, newToken)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to save edited file model", err)
	}

	return c.JSON(fileModel)
}
