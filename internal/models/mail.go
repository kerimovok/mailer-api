package models

import (
	"github.com/google/uuid"
	"github.com/kerimovok/go-pkg-database/sql"
)

type Attachment struct {
	sql.BaseModel
	MailID uuid.UUID `json:"mailId"`
	File   string    `json:"file"`
}

type Mail struct {
	sql.BaseModel
	To          string       `json:"to"`
	Subject     string       `json:"subject"`
	Template    string       `json:"template"`
	Data        sql.JSONB    `json:"data" gorm:"type:jsonb"`
	Status      string       `json:"status"`
	Error       string       `json:"error,omitempty"`
	Attachments []Attachment `json:"attachments"`
}
