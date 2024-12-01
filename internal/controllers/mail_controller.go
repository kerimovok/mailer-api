package controllers

import (
	"encoding/json"
	"log"
	"mailer-api/internal/models"
	"mailer-api/internal/workers"
	"mailer-api/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

type MailHandler struct {
	db     *gorm.DB
	client *asynq.Client
}

func NewMailHandler(db *gorm.DB, client *asynq.Client) *MailHandler {
	return &MailHandler{
		db:     db,
		client: client,
	}
}

func (h *MailHandler) SendMail(c *fiber.Ctx) error {
	var req models.MailRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to marshal data", err)
	}

	// Create mail record
	mail := models.Mail{
		To:       req.To,
		Subject:  req.Subject,
		Template: req.Template,
		Data:     string(dataJSON),
		Status:   "pending",
	}

	// Begin transaction
	tx := h.db.Begin()
	if err := tx.Create(&mail).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to create mail record: %v", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create mail record", err)
	}

	// Create attachment records
	for _, attachment := range req.Attachments {
		att := models.Attachment{
			MailID: mail.ID,
			File:   attachment.File,
		}
		if err := tx.Create(&att).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to create attachment record: %v", err)
			return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create attachment record", err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to commit transaction", err)
	}

	task, err := workers.NewEmailTask(mail.ID)
	if err != nil {
		log.Printf("Failed to create email task: %v", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create email task", err)
	}

	_, err = h.client.Enqueue(task)
	if err != nil {
		log.Printf("Failed to enqueue email task: %v", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to enqueue email task", err)
	}

	return utils.SuccessResponse(c, "Email queued successfully", mail)
}

func (h *MailHandler) GetMails(c *fiber.Ctx) error {
	var mails []models.Mail
	if err := h.db.Find(&mails).Error; err != nil {
		log.Printf("Failed to fetch mails: %v", err)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch mails", err)
	}
	return utils.SuccessResponse(c, "Mails fetched successfully", mails)
}

func (h *MailHandler) GetMailByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var mail models.Mail
	if err := h.db.First(&mail, id).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Mail not found", err)
	}
	return utils.SuccessResponse(c, "Mail fetched successfully", mail)
}
