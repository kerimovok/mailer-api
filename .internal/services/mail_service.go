package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"mailer-api/.internal/models"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
)

type MailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	smtpFrom     string
	dialer       *gomail.Dialer
	tempDir      string
	downloader   *AttachmentDownloader
}

func NewMailService(host, port, username, password, from string) *MailService {
	portInt, _ := strconv.Atoi(port)
	dialer := gomail.NewDialer(host, portInt, username, password)
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// Create temp directory if it doesn't exist
	tempDir := filepath.Join(os.TempDir(), "mailer-temp-attachments")
	os.MkdirAll(tempDir, 0755)

	return &MailService{
		smtpHost:     host,
		smtpPort:     port,
		smtpUsername: username,
		smtpPassword: password,
		smtpFrom:     from,
		dialer:       dialer,
		tempDir:      tempDir,
		downloader:   NewAttachmentDownloader(tempDir),
	}
}

func (s *MailService) SendMail(to, subject, templateName string, data string, attachments []models.AttachmentRequest) error {
	var templateData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &templateData); err != nil {
		return err
	}

	templatePath := filepath.Join("templates", templateName+".html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, templateData); err != nil {
		return err
	}

	smtpFrom := fmt.Sprintf("%s <%s>", s.smtpFrom, s.smtpUsername)

	m := gomail.NewMessage()
	m.SetHeader("From", smtpFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body.String())

	// Temporary files to clean up
	var tempFiles []string
	defer func() {
		// Clean up temporary files
		for _, file := range tempFiles {
			os.Remove(file)
		}
	}()

	// Process attachments
	var remoteUrls []string
	for _, attachment := range attachments {
		if isValidURL(attachment.FilePath) {
			remoteUrls = append(remoteUrls, attachment.FilePath)
		}
	}

	// Download remote files concurrently
	var downloadedFiles map[string]string // original URL -> downloaded path
	if len(remoteUrls) > 0 {
		files, err := s.downloader.DownloadFiles(remoteUrls)
		if err != nil {
			return fmt.Errorf("failed to download attachments: %w", err)
		}

		downloadedFiles = make(map[string]string)
		for i, url := range remoteUrls {
			downloadedFiles[url] = files[i]
			tempFiles = append(tempFiles, files[i])
		}
	}

	// Add attachments to email
	for _, attachment := range attachments {
		filePath := attachment.FilePath

		// If it's a remote file, use the downloaded version
		if isValidURL(filePath) {
			if downloadedPath, ok := downloadedFiles[filePath]; ok {
				filePath = downloadedPath
			} else {
				return fmt.Errorf("missing downloaded file for %s", filePath)
			}
		}

		m.Attach(filePath, gomail.Rename(attachment.FileName))
	}

	return s.dialer.DialAndSend(m)
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}
