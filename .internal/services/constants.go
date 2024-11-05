package services

const (
	MaxFileSize = 10 * 1024 * 1024 // 10MB
)

var AllowedMimeTypes = map[string]bool{
	// Document types
	"application/pdf":               true,
	"application/msword":            true,
	"application/vnd.ms-excel":      true,
	"application/vnd.ms-powerpoint": true,
	"application/vnd.ms-publisher":  true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/rtf":  true,
	"application/json": true,
	"application/xml":  true,

	// Image types
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/bmp":     true,
	"image/tiff":    true,
	"image/webp":    true,
	"image/svg+xml": true,

	// Audio types
	"audio/mpeg":  true,
	"audio/wav":   true,
	"audio/ogg":   true,
	"audio/mp4":   true,
	"audio/x-wav": true,

	// Video types
	"video/mp4":        true,
	"video/x-matroska": true,
	"video/quicktime":  true,
	"video/x-msvideo":  true,
	"video/webm":       true,
	"video/avi":        true,
	"video/mkv":        true,

	// Archive types
	"application/zip":              true,
	"application/x-rar-compressed": true,
	"application/x-tar":            true,
	"application/gzip":             true,
	"application/x-7z-compressed":  true,

	// Text types
	"text/plain": true,
	"text/html":  true,

	// Miscellaneous
	"application/x-shockwave-flash":                    true,
	"application/octet-stream":                         true,
	"application/x-msdownload":                         true,
	"application/x-apple-diskimage":                    true,
	"application/vnd.ms-excel.sheet.macroenabled.12":   true,
	"application/vnd.ms-word.document.macroenabled.12": true,
}
