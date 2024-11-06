package handlers

import (
	"encoding/json"
	"mailer-api/.internal/models"
	"mailer-api/.internal/workers"

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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Create attachment records
	for _, attachment := range req.Attachments {
		att := models.Attachment{
			MailID: mail.ID,
			File:   attachment.File,
		}
		if err := tx.Create(&att).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	task, err := workers.NewEmailTask(mail.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	_, err = h.client.Enqueue(task)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    mail,
	})
}

func (h *MailHandler) GetMails(c *fiber.Ctx) error {
	var mails []models.Mail
	if err := h.db.Find(&mails).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch mails"})
	}
	return c.JSON(mails)
}

func (h *MailHandler) GetMailByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var mail models.Mail
	if err := h.db.First(&mail, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Mail not found"})
	}
	return c.JSON(mail)
}
