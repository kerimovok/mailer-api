package workers

import (
	"context"
	"encoding/json"
	"mailer-api/.internal/models"
	"mailer-api/.internal/services"

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

func NewEmailTask(mailID uint) (*asynq.Task, error) {
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

	mailID := uint(payload["mail_id"].(float64))
	var mail models.Mail
	if err := processor.db.Preload("Attachments").First(&mail, mailID).Error; err != nil {
		return err
	}

	// Convert attachments to AttachmentRequest
	attachments := make([]models.AttachmentRequest, len(mail.Attachments))
	for i, att := range mail.Attachments {
		attachments[i] = models.AttachmentRequest{
			FileName: att.FileName,
			FilePath: att.FilePath,
		}
	}

	err := processor.service.SendMail(mail.To, mail.Subject, mail.Template, mail.Data, attachments)
	if err != nil {
		mail.Status = "failed"
		mail.Error = err.Error()
	} else {
		mail.Status = "sent"
	}

	return processor.db.Save(&mail).Error
}
