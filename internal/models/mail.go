package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Attachment struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	MailID    uuid.UUID      `json:"mailId"`
	File      string         `json:"file"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

type Mail struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	To          string         `json:"to"`
	Subject     string         `json:"subject"`
	Template    string         `json:"template"`
	Data        string         `json:"data" gorm:"type:jsonb"`
	Status      string         `json:"status"`
	Error       string         `json:"error,omitempty"`
	Attachments []Attachment   `json:"attachments"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

type MailRequest struct {
	To          string                 `json:"to"`
	Subject     string                 `json:"subject"`
	Template    string                 `json:"template"`
	Data        map[string]interface{} `json:"data"`
	Attachments []AttachmentRequest    `json:"attachments,omitempty"`
}

type AttachmentRequest struct {
	File string `json:"file"`
}
