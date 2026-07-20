package controller

import (
	"app/src/response"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
)

type UploadController struct{}

func NewUploadController() *UploadController {
	return &UploadController{}
}

func (ctrl *UploadController) UploadFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "No file uploaded")
	}

	// Ensure directory exists
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create upload directory")
	}

	// Generate safe unique filename
	ext := filepath.Ext(file.Filename)
	uniqueName := fmt.Sprintf("%s-%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, uniqueName)

	if err := c.SaveFile(file, filePath); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to save file: "+err.Error())
	}

	// Return URL path
	fileURL := fmt.Sprintf("/uploads/%s", uniqueName)

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[string]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "File uploaded successfully",
			Data:    fileURL,
		})
}
