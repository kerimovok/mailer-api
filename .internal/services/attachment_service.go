package services

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AttachmentDownloader struct {
	tempDir     string
	client      *http.Client
	maxFileSize int64
}

type DownloadResult struct {
	FilePath string
	Error    error
}

func NewAttachmentDownloader(tempDir string) *AttachmentDownloader {
	return &AttachmentDownloader{
		tempDir:     tempDir,
		client:      &http.Client{},
		maxFileSize: MaxFileSize,
	}
}

func (d *AttachmentDownloader) validateFileType(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	mimeType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("invalid content type: %w", err)
	}

	if !AllowedMimeTypes[mimeType] {
		return fmt.Errorf("unsupported file type: %s", mimeType)
	}
	return nil
}

func (d *AttachmentDownloader) DownloadFiles(urls []string) ([]string, error) {
	results := make(chan DownloadResult, len(urls))
	var wg sync.WaitGroup

	// Start concurrent downloads
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			filePath, err := d.downloadWithRetry(url, 3) // 3 retries
			results <- DownloadResult{FilePath: filePath, Error: err}
		}(url)
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var downloadedFiles []string
	var errors []error

	for result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else {
			downloadedFiles = append(downloadedFiles, result.FilePath)
		}
	}

	if len(errors) > 0 {
		return downloadedFiles, fmt.Errorf("some downloads failed: %v", errors)
	}

	return downloadedFiles, nil
}

func (d *AttachmentDownloader) downloadWithRetry(url string, retries int) (string, error) {
	var lastErr error
	for i := 0; i < retries; i++ {
		filePath, err := d.downloadFile(url)
		if err == nil {
			return filePath, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("failed after %d retries: %w", retries, lastErr)
}

func (d *AttachmentDownloader) downloadFile(url string) (string, error) {
	resp, err := d.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	if err := d.validateFileType(resp); err != nil {
		return "", err
	}

	contentLength := resp.ContentLength
	if contentLength > d.maxFileSize {
		return "", errors.New("file too large")
	}

	// Create temporary file
	tempFile := filepath.Join(d.tempDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(url)))
	out, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Use io.Copy with a LimitedReader for memory efficiency
	_, err = io.Copy(out, io.LimitReader(resp.Body, d.maxFileSize))
	if err != nil {
		os.Remove(tempFile)
		return "", err
	}

	return tempFile, nil
}
