package handlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	MaxFileSize      = 5 * 1024 * 1024 // 5MB
	UploadDir        = "./uploads"
	AllowedImageExts = ".jpg,.jpeg,.png,.gif,.webp"
	AllowedVideoExts = ".mp4,.webm,.mov"
	AllowedAudioExts = ".mp3,.wav,.ogg,.m4a"
	AllowedFileExts  = ".pdf,.doc,.docx,.txt,.zip"
)

// UploadFile handles file uploads
func UploadFile(c *fiber.Ctx) error {
	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "No file uploaded",
		})
	}

	// Check file size (5MB limit)
	if file.Size > MaxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   fmt.Sprintf("File size exceeds limit of 5MB (uploaded: %.2fMB)", float64(file.Size)/(1024*1024)),
		})
	}

	// Get file type from query parameter
	fileType := c.Query("type", "file") // image, video, audio, file
	if fileType != "image" && fileType != "video" && fileType != "audio" && fileType != "file" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid file type. Must be: image, video, audio, or file",
		})
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isAllowedExtension(ext, fileType) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   fmt.Sprintf("File extension %s not allowed for type %s", ext, fileType),
		})
	}

	// Create upload directory if not exists
	uploadPath := filepath.Join(UploadDir, fileType+"s")
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create upload directory",
		})
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s-%d%s", uuid.New().String(), time.Now().Unix(), ext)
	fullPath := filepath.Join(uploadPath, filename)

	// Save file
	if err := c.SaveFile(file, fullPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to save file",
		})
	}

	// Generate URL
	fileURL := fmt.Sprintf("/uploads/%ss/%s", fileType, filename)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"filename": file.Filename,
			"size":     file.Size,
			"type":     fileType,
			"url":      fileURL,
		},
	})
}

// UploadAvatar handles avatar uploads (separate endpoint with specific validation)
func UploadAvatar(c *fiber.Ctx) error {
	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "No avatar uploaded",
		})
	}

	// Check file size (2MB limit for avatars)
	if file.Size > 2*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Avatar size exceeds limit of 2MB",
		})
	}

	// Validate image extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !strings.Contains(AllowedImageExts, ext) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid image format. Allowed: jpg, jpeg, png, gif, webp",
		})
	}

	// Create avatars directory
	uploadPath := filepath.Join(UploadDir, "avatars")
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create upload directory",
		})
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s-%d%s", uuid.New().String(), time.Now().Unix(), ext)
	fullPath := filepath.Join(uploadPath, filename)

	// Save file
	if err := c.SaveFile(file, fullPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to save avatar",
		})
	}

	// Generate URL
	avatarURL := fmt.Sprintf("/uploads/avatars/%s", filename)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"url": avatarURL,
		},
	})
}

// isAllowedExtension checks if file extension is allowed for the given type
func isAllowedExtension(ext, fileType string) bool {
	switch fileType {
	case "image":
		return strings.Contains(AllowedImageExts, ext)
	case "video":
		return strings.Contains(AllowedVideoExts, ext)
	case "audio":
		return strings.Contains(AllowedAudioExts, ext)
	case "file":
		return strings.Contains(AllowedFileExts, ext)
	default:
		return false
	}
}

// GetFile serves uploaded files
func GetFile(c *fiber.Ctx) error {
	fileType := c.Params("type")
	filename := c.Params("filename")

	// Validate file type
	if fileType != "images" && fileType != "videos" && fileType != "audios" && fileType != "files" && fileType != "avatars" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid file type",
		})
	}

	// Construct file path
	filePath := filepath.Join(UploadDir, fileType, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "File not found",
		})
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to open file",
		})
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get file info",
		})
	}

	// Set content type based on extension
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := getContentType(ext)
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream file to client
	_, err = io.Copy(c.Response().BodyWriter(), file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to send file",
		})
	}

	return nil
}

// getContentType returns content type based on file extension
func getContentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
