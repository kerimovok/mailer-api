package handlers

import (
	"log"
	"mailer-api/internal/database"
	"mailer-api/internal/models"
	"mailer-api/internal/requests"
	"mailer-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/kerimovok/go-pkg-utils/httpx"
	"github.com/kerimovok/go-pkg-utils/validator"
)

func SendMail(c *fiber.Ctx) error {
	var input requests.MailRequest
	if err := c.BodyParser(&input); err != nil {
		response := httpx.BadRequest("Invalid request body", err)
		return httpx.SendResponse(c, response)
	}

	validationErrors := validator.ValidateStruct(&input)
	if validationErrors.HasErrors() {
		// Convert validator.ValidationErrors to []httpx.ValidationError
		httpxErrors := make([]httpx.ValidationError, len(validationErrors))
		for i, err := range validationErrors {
			httpxErrors[i] = httpx.ValidationError{
				Field:   err.Field,
				Message: err.Message,
			}
		}
		response := httpx.UnprocessableEntityWithValidation("Validation failed", httpxErrors)
		return httpx.SendValidationResponse(c, response)
	}

	// Use unified email processing
	mail, err := services.ProcessEmailRequest(input.To, input.Subject, input.Template, input.Data, input.Attachments)
	if err != nil {
		log.Printf("failed to process mail: %v", err)
		response := httpx.InternalServerError("Failed to process mail", err)
		return httpx.SendResponse(c, response)
	}

	response := httpx.OK("Email processed successfully", mail)
	return httpx.SendResponse(c, response)
}

func GetMails(c *fiber.Ctx) error {
	var mails []models.Mail
	if err := database.DB.Find(&mails).Error; err != nil {
		log.Printf("failed to fetch mails: %v", err)
		response := httpx.InternalServerError("Failed to fetch mails", err)
		return httpx.SendResponse(c, response)
	}
	response := httpx.OK("Mails fetched successfully", mails)
	return httpx.SendResponse(c, response)
}

func GetMailByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var mail models.Mail
	if err := database.DB.First(&mail, id).Error; err != nil {
		response := httpx.NotFound("Mail not found")
		return httpx.SendResponse(c, response)
	}
	response := httpx.OK("Mail fetched successfully", mail)
	return httpx.SendResponse(c, response)
}
