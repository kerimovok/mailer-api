package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"mailer-api/internal/requests"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kerimovok/go-pkg-utils/config"
	"github.com/kerimovok/go-pkg-utils/errors"
	"gopkg.in/gomail.v2"
)

var (
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	smtpFrom     string
	dialer       *gomail.Dialer
)

func InitMailService() {
	smtpHost = config.GetEnv("SMTP_HOST")
	smtpPort = config.GetEnv("SMTP_PORT")
	smtpUsername = config.GetEnv("SMTP_USERNAME")
	smtpPassword = config.GetEnv("SMTP_PASSWORD")
	smtpFrom = config.GetEnv("SMTP_FROM")

	portInt, _ := strconv.Atoi(smtpPort)
	dialer = gomail.NewDialer(smtpHost, portInt, smtpUsername, smtpPassword)
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
}

func createTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
	}
}

func SendMail(to, subject, templateName string, data string, attachments []requests.AttachmentRequest) error {
	var templateData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &templateData); err != nil {
		return errors.InternalError("UNMARSHAL_DATA", "Failed to unmarshal template data").WithMetadata("error", err.Error())
	}

	// Parse subject as template
	subjectTmpl, err := template.New("subject").Parse(subject)
	if err != nil {
		return errors.InternalError("PARSE_SUBJECT", "Failed to parse subject template").WithMetadata("error", err.Error())
	}

	var parsedSubject bytes.Buffer
	if err := subjectTmpl.Execute(&parsedSubject, templateData); err != nil {
		return errors.InternalError("EXECUTE_SUBJECT", "Failed to execute subject template").WithMetadata("error", err.Error())
	}

	// Use absolute path for template
	templatePath := filepath.Join("templates", templateName+".html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		log.Printf("Template file not found: %s", templatePath)
		return errors.NotFoundError("TEMPLATE_NOT_FOUND", "Template file not found").WithMetadata("template", templateName)
	}

	// Create template with function map
	tmpl, err := template.New(templateName + ".html").
		Funcs(createTemplateFuncMap()).
		ParseFiles(templatePath)
	if err != nil {
		return errors.InternalError("PARSE_TEMPLATE", "Failed to parse template").WithMetadata("error", err.Error())
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, templateData); err != nil {
		return errors.InternalError("EXECUTE_TEMPLATE", "Failed to execute template").WithMetadata("error", err.Error())
	}

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", smtpFrom, smtpUsername))
	m.SetHeader("To", to)
	m.SetHeader("Subject", parsedSubject.String())
	m.SetBody("text/html", body.String())

	// Process attachments
	for _, attachment := range attachments {
		attachPath := filepath.Join("attachments", attachment.File)
		if _, err := os.Stat(attachPath); os.IsNotExist(err) {
			return errors.NotFoundError("ATTACHMENT_NOT_FOUND", "Attachment file not found").WithMetadata("file", attachment.File)
		}
		m.Attach(attachPath)
	}

	if err := dialer.DialAndSend(m); err != nil {
		return errors.InternalError("SEND_EMAIL", "Failed to send email").WithMetadata("error", err.Error())
	}

	return nil
}
