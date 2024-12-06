package requests

type MailRequest struct {
	To          string                 `json:"to" validate:"required,email"`
	Subject     string                 `json:"subject" validate:"required"`
	Template    string                 `json:"template" validate:"required"`
	Data        map[string]interface{} `json:"data" validate:"required"`
	Attachments []AttachmentRequest    `json:"attachments,omitempty"`
}

type AttachmentRequest struct {
	File string `json:"file" validate:"required"`
}
