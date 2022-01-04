package handlers

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/thetkpark/cscms-temp-storage/data"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"github.com/thetkpark/cscms-temp-storage/service/storage"
	"github.com/thetkpark/cscms-temp-storage/service/token"
	"go.uber.org/zap"
	"regexp"
	"strconv"
	"time"
)

type ImageRouteHandler struct {
	log               *zap.SugaredLogger
	imageDataStore    data.ImageDataStore
	imageStoreManager storage.ImageManager
	tokenManager      token.Manager
}

func NewImageRouteHandler(log *zap.SugaredLogger, imgDataStore data.ImageDataStore, store storage.ImageManager, token token.Manager) *ImageRouteHandler {
	return &ImageRouteHandler{
		log:               log,
		imageDataStore:    imgDataStore,
		imageStoreManager: store,
		tokenManager:      token,
	}
}

// UploadImage handlers
// @Summary Upload new image
// @Description Upload new image
// @Tags Image
// @Accept  multipart/form-data
// @Produce  json
// @Param       image  formData  file  true  "Image"
// @Success      201  {object}  model.Image
// @Failure      400  {object}  handlers.ErrorResponse
// @Failure      413  {object}  handlers.ErrorResponse
// @Failure      500  {object}  handlers.ErrorResponse
// @Router /api/image [post]
func (h *ImageRouteHandler) UploadImage(c *fiber.Ctx) error {
	// Get image file from Form
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusBadRequest, "Unable to get file from multipart/form-data", err)
	}

	// Check image size (5MB)
	if fileHeader.Size > 5<<20 {
		return NewHTTPError(h.log, fiber.StatusRequestEntityTooLarge, "Image file too large", nil)
	}
	// Check image format and get extension
	fileExtension, err := h.validateFileFormat(fileHeader.Header.Get("Content-Type"), fileHeader.Filename)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusBadRequest, "Invalid file extension", err)
	}
	imageToken, err := h.tokenManager.GenerateImageToken()
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusBadRequest, "Unable to generate image token", err)
	}
	imagePath := fmt.Sprintf("%s.%s", imageToken, fileExtension)

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "Unable to open file from the fileHeader", err)
	}

	// Upload the image to storage
	if err := h.imageStoreManager.UploadImage(imagePath, file); err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "Unable to upload image", err)
	}

	// Create ImageInfo struct
	imageInfo := &model.Image{
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		OriginalFilename: fileHeader.Filename,
		FileSize:         uint64(fileHeader.Size),
		FilePath:         imagePath,
	}

	// Get userId if exist
	user := c.UserContext().Value("user")
	if user != nil {
		userModel, ok := user.(*model.User)
		if !ok {
			return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse to user model", fmt.Errorf("user model convertion error"))
		}
		imageInfo.UserID = userModel.ID
	}

	// Save image info to db
	err = h.imageDataStore.Create(imageInfo)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to save image info to db", err)
	}

	// Return the image info
	return c.Status(fiber.StatusCreated).JSON(imageInfo)
}

// GetOwnImages handlers
// @Summary List uploaded images
// @Description List uploaded images from the user
// @Tags Image
// @Produce  json
// @Success      200  {array}  model.Image
// @Failure      400  {object}  handlers.ErrorResponse
// @Failure      500  {object}  handlers.ErrorResponse
// @Router /api/image [get]
func (h *ImageRouteHandler) GetOwnImages(c *fiber.Ctx) error {
	// Get userId
	user := c.UserContext().Value("user")
	userModel, ok := user.(*model.User)
	if !ok {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse to user model", fmt.Errorf("user model convertion error"))
	}

	images, err := h.imageDataStore.FindByUserID(userModel.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "Unable to find images by user ID", err)
	}

	return c.JSON(images)
}

func (h *ImageRouteHandler) IsOwnImage(c *fiber.Ctx) error {
	imageId := c.Params("imageID", "")
	if len(imageId) == 0 {
		return NewHTTPError(h.log, fiber.StatusBadRequest, "Image ID must be provided", nil)
	}
	imageIDInt, err := strconv.Atoi(imageId)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusBadRequest, "Image ID must be integer", nil)
	}

	// Get userId
	user := c.UserContext().Value("user")
	userModel, ok := user.(*model.User)
	if !ok {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse to user model", fmt.Errorf("user model convertion error"))
	}

	image, err := h.imageDataStore.FindByID(uint(imageIDInt))
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "Unable to query image", err)
	}
	if image == nil {
		return NewHTTPError(h.log, fiber.StatusNotFound, "Image not found", nil)
	}
	if image.UserID != userModel.ID {
		return NewHTTPError(h.log, fiber.StatusForbidden, "Forbidden", nil)
	}

	c.SetUserContext(context.WithValue(c.UserContext(), "image", image))
	return c.Next()
}

// DeleteImage handlers
// @Summary Delete image
// @Description Delete uploaded image from the user
// @Tags Image
// @Produce  json
// @Param        imageID       path      int      true  "Image ID"
// @Success      200  {object}  model.Image
// @Failure      400  {object}  handlers.ErrorResponse
// @Failure      401  {object}  handlers.ErrorResponse
// @Failure      403  {object}  handlers.ErrorResponse
// @Failure      500  {object}  handlers.ErrorResponse
// @Router /api/image/{imageID} [delete]
func (h *ImageRouteHandler) DeleteImage(c *fiber.Ctx) error {
	image, ok := c.UserContext().Value("image").(*model.Image)
	if !ok {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to parse image model", fmt.Errorf("unable to parse image model"))
	}

	// Delete image record in db
	err := h.imageDataStore.DeleteByID(image.ID)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to delete image in db", err)
	}

	// Delete image on storage
	err = h.imageStoreManager.DeleteImage(image.FilePath)
	if err != nil {
		return NewHTTPError(h.log, fiber.StatusInternalServerError, "unable to delete image on storage", err)
	}

	return c.JSON(image)
}

func (h *ImageRouteHandler) validateFileFormat(mimeType string, fileName string) (string, error) {
	switch mimeType {
	case "image/png", "image/jpeg", "image/gif", "image/x-icon", "image/heic", "image/webp", "image/tiff", "image/svg+xml", "image/bmp", "image/apng", "image/avif":
		return h.getFileExtension(fileName), nil
	default:
		return "", fmt.Errorf("%s is not supported", mimeType)
	}

}

func (h *ImageRouteHandler) getFileExtension(fileName string) string {
	regex := regexp.MustCompile(".+\\.(.+)")
	group := regex.FindStringSubmatch(fileName)
	return group[1]
}
