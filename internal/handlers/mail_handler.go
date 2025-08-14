package handlers

import (
	"encoding/json"
	"log"
	"mailer-api/internal/database"
	"mailer-api/internal/models"
	"mailer-api/internal/requests"
	"mailer-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/kerimovok/go-pkg-database/sql"
	"github.com/kerimovok/go-pkg-utils/httpx"
	"github.com/kerimovok/go-pkg-utils/validator"
	"gorm.io/gorm"
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

	// Create mail record
	mail := models.Mail{
		To:       input.To,
		Subject:  input.Subject,
		Template: input.Template,
		Data:     sql.JSONB(input.Data),
		Status:   "pending",
	}

	// Use WithTransaction helper
	err := sql.WithTransaction(database.DB, func(tx *gorm.DB) error {
		// Create mail record
		if err := tx.Create(&mail).Error; err != nil {
			return err
		}

		// Create attachment records
		for _, attachment := range input.Attachments {
			att := models.Attachment{
				MailID: mail.ID,
				File:   attachment.File,
			}
			if err := tx.Create(&att).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("transaction failed: %v", err)
		response := httpx.InternalServerError("Failed to process mail", err)
		return httpx.SendResponse(c, response)
	}

	// Convert data to string for SendMail
	dataStr, err := json.Marshal(input.Data)
	if err != nil {
		log.Printf("failed to marshal data: %v", err)
		response := httpx.InternalServerError("Failed to marshal data", err)
		return httpx.SendResponse(c, response)
	}

	// Send the email immediately
	err = services.SendMail(input.To, input.Subject, input.Template, string(dataStr), input.Attachments)
	if err != nil {
		log.Printf("failed to send mail: %v", err)
		mail.Status = "failed"
		mail.Error = err.Error()
	} else {
		log.Printf("mail sent successfully: %s", mail.ID.String())
		mail.Status = "sent"
	}

	// Update mail status
	if err := database.DB.Save(&mail).Error; err != nil {
		log.Printf("failed to update mail status: %v", err)
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
