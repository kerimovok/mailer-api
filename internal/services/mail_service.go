package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"mailer-api/internal/requests"
	"mailer-api/pkg/utils"
	"os"
	"path/filepath"
	"strconv"

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
	smtpHost = utils.GetEnv("SMTP_HOST")
	smtpPort = utils.GetEnv("SMTP_PORT")
	smtpUsername = utils.GetEnv("SMTP_USERNAME")
	smtpPassword = utils.GetEnv("SMTP_PASSWORD")
	smtpFrom = utils.GetEnv("SMTP_FROM")

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
		utils.LogWarn("Template file not found: " + templatePath)
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
	m.SetHeader("From", fmt.Sprintf("%s <%s>", smtpFrom, smtpUsername))
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

	if err := dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
