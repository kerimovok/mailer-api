package models

import (
	"time"

	"gorm.io/gorm"
)

type Attachment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	MailID    uint           `json:"mailId"`
	File      string         `json:"file"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

type Mail struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
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
