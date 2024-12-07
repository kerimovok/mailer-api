package handlers

import (
	"encoding/json"
	"mailer-api/internal/config"
	"mailer-api/internal/models"
	"mailer-api/internal/requests"
	"mailer-api/internal/workers"
	"mailer-api/pkg/database"
	"mailer-api/pkg/utils"
	"mailer-api/pkg/validator"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SendMail(c *fiber.Ctx) error {
	var input requests.MailRequest
	if err := c.BodyParser(&input); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if err := validator.ValidateStruct(&input); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	dataJSON, err := json.Marshal(input.Data)
	if err != nil {
		utils.LogError("failed to marshal data", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to marshal data", err)
	}

	// Create mail record
	mail := models.Mail{
		To:       input.To,
		Subject:  input.Subject,
		Template: input.Template,
		Data:     string(dataJSON),
		Status:   "pending",
	}

	// Use WithTransaction helper
	err = database.WithTransaction(func(tx *gorm.DB) error {
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
		utils.LogError("transaction failed", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to process mail", err)
	}

	task, err := workers.NewEmailTask(mail.ID)
	if err != nil {
		utils.LogError("failed to create email task", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create email task", err)
	}

	_, err = config.AsynqClient.Enqueue(task)
	if err != nil {
		utils.LogError("failed to enqueue email task", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to enqueue email task", err)
	}

	return utils.SuccessResponse(c, "Email queued successfully", mail)
}

func GetMails(c *fiber.Ctx) error {
	var mails []models.Mail
	if err := database.DB.Find(&mails).Error; err != nil {
		utils.LogError("failed to fetch mails", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch mails", err)
	}
	return utils.SuccessResponse(c, "Mails fetched successfully", mails)
}

func GetMailByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var mail models.Mail
	if err := database.DB.First(&mail, id).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Mail not found", err)
	}
	return utils.SuccessResponse(c, "Mail fetched successfully", mail)
}
