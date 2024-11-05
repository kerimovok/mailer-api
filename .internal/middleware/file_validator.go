package middleware

import (
	"mailer-api/.internal/services"

	"github.com/gofiber/fiber/v2"
)

func ValidateFileUpload() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if there's a file being uploaded
		file, err := c.FormFile("file")
		if err != nil {
			return c.Next()
		}

		// Check file size
		if file.Size > services.MaxFileSize {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "File size exceeds maximum limit",
			})
		}

		// Get file extension and check mime type
		contentType := file.Header.Get("Content-Type")
		if !services.AllowedMimeTypes[contentType] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Unsupported file type",
			})
		}

		return c.Next()
	}
}
