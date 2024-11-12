package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"mailer-api/.internal/models"
	"mailer-api/.internal/services"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

type MailProcessor struct {
	db      *gorm.DB
	service *services.MailService
}

func NewMailProcessor(db *gorm.DB, service *services.MailService) *MailProcessor {
	return &MailProcessor{
		db:      db,
		service: service,
	}
}

func NewEmailTask(mailID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(map[string]interface{}{"mail_id": mailID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTypeSendEmail, payload), nil
}

func (processor *MailProcessor) ProcessMail(ctx context.Context, t *asynq.Task) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	mailIDInterface, ok := payload["mail_id"]
	if !ok {
		return fmt.Errorf("mail_id not found in payload")
	}

	mailIDStr, ok := mailIDInterface.(string)
	if !ok {
		return fmt.Errorf("mail_id is not a string")
	}

	mailID, err := uuid.Parse(mailIDStr)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	var mail models.Mail
	if err := processor.db.Preload("Attachments").First(&mail, mailID).Error; err != nil {
		return err
	}

	// Convert attachments to AttachmentRequest
	attachments := make([]models.AttachmentRequest, len(mail.Attachments))
	for i, att := range mail.Attachments {
		attachments[i] = models.AttachmentRequest{
			File: att.File,
		}
	}

	err = processor.service.SendMail(mail.To, mail.Subject, mail.Template, mail.Data, attachments)
	if err != nil {
		mail.Status = "failed"
		mail.Error = err.Error()
	} else {
		mail.Status = "sent"
	}

	return processor.db.Save(&mail).Error
}
