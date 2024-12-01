package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"mailer-api/internal/models"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/gomail.v2"
)

type MailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	smtpFrom     string
	dialer       *gomail.Dialer
}

func NewMailService(host, port, username, password, from string) *MailService {
	portInt, _ := strconv.Atoi(port)
	dialer := gomail.NewDialer(host, portInt, username, password)
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return &MailService{
		smtpHost:     host,
		smtpPort:     port,
		smtpUsername: username,
		smtpPassword: password,
		smtpFrom:     from,
		dialer:       dialer,
	}
}

func createTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
	}
}

func (s *MailService) SendMail(to, subject, templateName string, data string, attachments []models.AttachmentRequest) error {
	var templateData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &templateData); err != nil {
		return fmt.Errorf("failed to unmarshal template data: %w", err)
	}

	// Parse subject as template
	subjectTmpl, err := template.New("subject").Parse(subject)
	if err != nil {
		return fmt.Errorf("failed to parse subject template: %w", err)
	}

	var parsedSubject bytes.Buffer
	if err := subjectTmpl.Execute(&parsedSubject, templateData); err != nil {
		return fmt.Errorf("failed to execute subject template: %w", err)
	}

	// Use absolute path for template
	templatePath := filepath.Join("templates", templateName+".html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		log.Printf("Template file not found: %s", templatePath)
		return fmt.Errorf("template not found: %s", templateName)
	}

	// Create template with function map
	tmpl, err := template.New(templateName + ".html").
		Funcs(createTemplateFuncMap()).
		ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.smtpFrom, s.smtpUsername))
	m.SetHeader("To", to)
	m.SetHeader("Subject", parsedSubject.String())
	m.SetBody("text/html", body.String())

	// Process attachments
	for _, attachment := range attachments {
		attachPath := filepath.Join("attachments", attachment.File)
		if _, err := os.Stat(attachPath); os.IsNotExist(err) {
			return fmt.Errorf("attachment file not found: %s", attachment.File)
		}
		m.Attach(attachPath)
	}

	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}