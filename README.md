# Mailer API

A robust email service built with Go, featuring asynchronous processing, template support, and advanced attachment handling.

## Features

-   Asynchronous email processing using Redis and Asynq
-   HTML email templates
-   Advanced file attachment handling:
    -   Local and remote file support
    -   Concurrent downloads for multiple attachments
    -   Automatic file type validation
    -   File size limits
    -   Memory-efficient streaming for large files
    -   Retry logic for failed downloads
-   PostgreSQL for data persistence
-   Docker containerization
-   Monitoring UI with Asynqmon
-   Database management with Adminer

## Tech Stack

-   Go 1.23.2
-   PostgreSQL 15
-   Redis 7
-   Fiber (Web Framework)
-   GORM (ORM)
-   Asynq (Task Queue)
-   Docker & Docker Compose

## Prerequisites

-   Docker and Docker Compose
-   Go 1.23.2 or higher
-   SMTP server credentials

## Configuration

Create a `.env` file in the root directory with the following variables:

```env
# Server
PORT=3000

# SMTP Configuration
SMTP_HOST=your-smtp-host
SMTP_PORT=587
SMTP_USERNAME=your-smtp-username
SMTP_PASSWORD=your-smtp-password
SMTP_FROM=Company Name

# If you are not using docker-compose, you need to set the following variables below

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=mailer

# Redis
REDIS_ADDR=redis:6379
```

## File Attachment Specifications

-   Maximum file size: 10MB
    -   Supported file types:
        -   Documents (.pdf, .doc, .docx, .xls, .xlsx, .ppt, .pptx, .rtf, .json, .xml)
        -   Images (.jpeg, .png, .gif, .bmp, .tiff, .webp, .svg)
        -   Audio (.mp3, .wav, .ogg, .mp4, .x-wav)
        -   Video (.mp4, .mkv, .avi, .mov, .webm)
        -   Archives (.zip, .rar, .tar, .gz, .7z)
        -   Text (.txt, .html)
        -   Miscellaneous (.swf, .bin, .exe, .dmg)

## API Endpoints

### Send Email

```http
POST /api/v1/mails
```

Request body:

```json
{
	"to": "recipient@example.com",
	"subject": "Email Subject",
	"template": "welcome",
	"data": {
		"name": "John Doe"
	},
	"attachments": [
		{
			"file_name": "document.pdf",
			"file_path": "https://example.com/document.pdf"
		},
		{
			"file_name": "local-file.jpg",
			"file_path": "/path/to/local/file.jpg"
		}
	]
}
```

### Get All Emails

```http
GET /api/v1/mails
```

### Get Email by ID

```http
GET /api/v1/mails/:id
```

## Project Structure

```
.
├── .internal/
│   ├── config/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── routes/
│   ├── services/
│   └── workers/
├── templates/
├── .env
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── main.go
```

## Running the Application

1. Clone the repository
2. Create and configure the `.env` file
3. Start the services:

```bash
docker-compose up -d
```

The API will be available at `http://localhost:3000`

## Monitoring

-   Asynqmon UI: `http://localhost:8080`
-   Adminer (Database Management): `http://localhost:8081`

## Development

To run the application locally for development:

```bash
go mod download
go run main.go
```

## License

[MIT License](LICENSE)
