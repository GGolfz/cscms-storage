package handlers

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/hashicorp/go-hclog"
	"github.com/thetkpark/cscms-temp-storage/data"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"github.com/thetkpark/cscms-temp-storage/service"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

type FileRoutesHandler struct {
	log               hclog.Logger
	encryptionManager service.EncryptionManager
	fileDataStore     data.FileDataStore
	storageManager    service.StorageManager
	maxStoreDuration  time.Duration
}

func NewFileRoutesHandler(log hclog.Logger, enc service.EncryptionManager, data data.FileDataStore, store service.StorageManager, duration time.Duration) *FileRoutesHandler {
	return &FileRoutesHandler{
		log:               log,
		encryptionManager: enc,
		fileDataStore:     data,
		storageManager:    store,
		maxStoreDuration:  duration,
	}
}

func (h *FileRoutesHandler) UploadFile(c *fiber.Ctx) error {
	tStart := time.Now()
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to get file from form-data", err)
	}

	// Create new fileId
	fileId, err := service.GenerateFileId()
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to create file id", err)
	}

	// Open multipart form header
	file, err := fileHeader.Open()
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to open file", err)
	}

	// Encrypt the file
	tEncrypt := time.Now()
	encrypted, nonce, err := h.encryptionManager.Encrypt(file)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable encrypt the file", err)
	}
	encryptFileDuration := time.Now().Sub(tEncrypt)

	// Write encrypted file to disk
	if err := h.storageManager.WriteToNewFile(fileId, encrypted); err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to write encrypted data to file", err)
	}

	// Check slug
	token, err := service.GenerateFileToken()
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to generate file token", err)
	}
	fileToken := strings.ToLower(c.Query("slug", token))
	// Check if slug is available
	existingFile, err := h.fileDataStore.FindByToken(fileToken)
	if existingFile != nil {
		return NewHTTPError(h.log, fiber.StatusBadRequest, fmt.Sprintf("%s slug is used", fileToken), nil)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to get existing file token", err)
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

	// Create new fileInfo record in db
	fileInfo := &model.File{
		ID:        fileId,
		Token:     fileToken,
		Nonce:     nonce,
		Filename:  fileHeader.Filename,
		FileSize:  uint64(fileHeader.Size),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		ExpiredAt: time.Now().UTC().Add(storeDuration),
		Visited:   0,
		UserID:    0,
	}

	// Get userId if exist
	user := c.UserContext().Value("user")
	if user != nil {
		userModel, ok := user.(*model.User)
		if !ok {
			return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse to user model", fmt.Errorf("user model convert error"))
		}
		fileInfo.UserID = userModel.ID
	}

	err = h.fileDataStore.Create(fileInfo)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to save file info to db", err)
	}

	return c.JSON(fiber.Map{
		"id":           fileInfo.ID,
		"token":        fileInfo.Token,
		"file_size":    fileInfo.FileSize,
		"file_name":    fileInfo.Filename,
		"created_at":   fileInfo.CreatedAt,
		"expired_at":   fileInfo.ExpiredAt,
		"encrypt_time": encryptFileDuration.String(),
		"total_time":   time.Since(tStart).String(),
	})
}

func (h *FileRoutesHandler) GetFile(c *fiber.Ctx) error {
	token := strings.ToLower(c.Params("token"))

	// Find file by token
	file, err := h.fileDataStore.FindByToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Redirect(c.BaseURL())
		}
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to get file query", err)
	}

	// Check if file still exist on storage
	if exist, err := h.storageManager.Exist(file.ID); !exist {
		if err == nil {
			// File is not exist anymore
			return c.Redirect(c.BaseURL())
		}
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to check if file exist", err)
	}

	// Get encrypted file from storage manager
	encryptedFile, err := h.storageManager.OpenFile(file.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to open encrypted file", err)
	}

	// Decrypt file
	err = h.encryptionManager.Decrypt(encryptedFile, file.Nonce, c)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to decrypt", err)
	}

	// Increase visited count
	err = h.fileDataStore.IncreaseVisited(file.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to increase count", err)
	}

	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Filename))
	c.Set("Content-Type", "application/octet-stream")

	return nil
}
