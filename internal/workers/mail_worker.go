package workers

import (
	"context"
	"encoding/json"
	"log"
	"mailer-api/internal/constants"
	"mailer-api/internal/database"
	"mailer-api/internal/models"
	"mailer-api/internal/requests"
	"mailer-api/internal/services"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/kerimovok/go-pkg-utils/errors"
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
		return nil, errors.InternalError("MARSHAL_PAYLOAD", "Failed to marshal email task payload").WithMetadata("error", err.Error())
	}
	return asynq.NewTask(constants.TaskTypeSendEmail, payload), nil
}

// ProcessMail processes a mail task
func ProcessMail(ctx context.Context, t *asynq.Task) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return errors.InternalError("UNMARSHAL_PAYLOAD", "Failed to unmarshal task payload").WithMetadata("error", err.Error())
	}

	mailIDInterface, ok := payload["mail_id"]
	if !ok {
		return errors.ValidationError("MISSING_MAIL_ID", "mail_id not found in payload")
	}

	mailIDStr, ok := mailIDInterface.(string)
	if !ok {
		return errors.ValidationError("INVALID_MAIL_ID", "mail_id must be a string")
	}

	mailID, err := uuid.Parse(mailIDStr)
	if err != nil {
		return errors.ValidationError("INVALID_UUID", "Invalid mail ID format").WithMetadata("error", err.Error())
	}

	var mail models.Mail
	if err := database.DB.Preload("Attachments").First(&mail, mailID).Error; err != nil {
		return errors.NotFoundError("MAIL_NOT_FOUND", "Mail not found").WithMetadata("mailID", mailID.String())
	}

	// Convert attachments to AttachmentRequest
	attachments := make([]requests.AttachmentRequest, len(mail.Attachments))
	for i, att := range mail.Attachments {
		attachments[i] = requests.AttachmentRequest{
			File: att.File,
		}
	}

	log.Printf("processing mail: %s", mail.ID.String())

	// Convert JSONB to string for the service
	dataStr, err := json.Marshal(mail.Data)
	if err != nil {
		return errors.InternalError("MARSHAL_DATA", "Failed to marshal mail data").WithMetadata("error", err.Error())
	}

	err = services.SendMail(mail.To, mail.Subject, mail.Template, string(dataStr), attachments)
	if err != nil {
		log.Printf("failed to send mail: %v", err)
		mail.Status = StatusFailed
		mail.Error = err.Error()
	} else {
		log.Printf("mail sent successfully: %s", mail.ID.String())
		mail.Status = StatusSent
	}

	if err := database.DB.Save(&mail).Error; err != nil {
		return errors.InternalError("UPDATE_MAIL", "Failed to update mail status").WithMetadata("error", err.Error())
	}

	return nil
}
