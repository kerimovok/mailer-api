package models

import (
	"time"

	"gorm.io/gorm"
)

type Attachment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	MailID    uint           `json:"mail_id"`
	FileName  string         `json:"file_name"`
	FilePath  string         `json:"file_path"`
	FileSize  int64          `json:"file_size"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
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
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type MailRequest struct {
	To          string                 `json:"to"`
	Subject     string                 `json:"subject"`
	Template    string                 `json:"template"`
	Data        map[string]interface{} `json:"data"`
	Attachments []AttachmentRequest    `json:"attachments,omitempty"`
}

type AttachmentRequest struct {
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
	FileSize int64  `json:"file_size,omitempty"`
}
