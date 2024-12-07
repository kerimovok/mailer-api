package workers

import (
	"context"
	"encoding/json"
	"mailer-api/internal/constants"
	"mailer-api/internal/models"
	"mailer-api/internal/requests"
	"mailer-api/internal/services"
	"mailer-api/pkg/database"
	"mailer-api/pkg/utils"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	StatusPending = "pending"
	StatusSent    = "sent"
	StatusFailed  = "failed"
)

// NewEmailTask creates a new email task with the given mail ID
func NewEmailTask(mailID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(map[string]interface{}{"mail_id": mailID})
	if err != nil {
		return nil, utils.WrapError("marshal email task payload", err)
	}
	return asynq.NewTask(constants.TaskTypeSendEmail, payload), nil
}

// ProcessMail processes a mail task
func ProcessMail(ctx context.Context, t *asynq.Task) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return utils.WrapError("unmarshal task payload", err)
	}

	mailIDInterface, ok := payload["mail_id"]
	if !ok {
		return utils.WrapError("get mail_id from payload", nil)
	}

	mailIDStr, ok := mailIDInterface.(string)
	if !ok {
		return utils.WrapError("convert mail_id to string", nil)
	}

	mailID, err := uuid.Parse(mailIDStr)
	if err != nil {
		return utils.WrapError("parse mail UUID", err)
	}

	var mail models.Mail
	if err := database.DB.Preload("Attachments").First(&mail, mailID).Error; err != nil {
		return utils.WrapError("fetch mail from database", err)
	}

	// Convert attachments to AttachmentRequest
	attachments := make([]requests.AttachmentRequest, len(mail.Attachments))
	for i, att := range mail.Attachments {
		attachments[i] = requests.AttachmentRequest{
			File: att.File,
		}
	}

	utils.LogInfo("processing mail: " + mail.ID.String())

	err = services.SendMail(mail.To, mail.Subject, mail.Template, mail.Data, attachments)
	if err != nil {
		utils.LogError("failed to send mail", err)
		mail.Status = StatusFailed
		mail.Error = err.Error()
	} else {
		utils.LogInfo("mail sent successfully: " + mail.ID.String())
		mail.Status = StatusSent
	}

	if err := database.DB.Save(&mail).Error; err != nil {
		return utils.WrapError("update mail status", err)
	}

	return nil
}
