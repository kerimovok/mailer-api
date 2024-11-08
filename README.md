# Mailer API

A robust email service built with Go, featuring asynchronous processing, template support, and local file attachment.

## Features

-   Asynchronous email processing using Redis and Asynq
-   HTML email templates
-   Local file attachment
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

## Security Notes

⚠️ **Important Security Considerations:**

1. In production environments, DO NOT expose the API port directly
2. The routes are not protected by authentication
3. Use a reverse proxy (like Nginx or Traefik) for production deployments
4. Ensure proper network security and access controls are in place

## Configuration

Create a `.env` file in the root directory with the following variables:

```env
#== SERVER ==#
PORT=3002

#== SMTP ==#
SMTP_HOST=smtp.example.com
SMTP_PORT=587 # 465, 587
SMTP_USERNAME=your-email@example.com
SMTP_PASSWORD=your-password
SMTP_FROM=Company Name # <username> will be appended to this, don't include it here

#== IF YOU DON'T WANT TO USE DOCKER COMPOSE ==#
#== YOU MUST PROVIDE OWN POSTGRES AND REDIS ==#
#== DOCKER COMPOSE WILL NOT USE THEM ==#

#== Postgres ==#
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=mailer

#== Redis ==#
REDIS_ADDR=redis:6379
REDIS_PASSWORD=redis
```

## Templates

Create a `templates` directory in the root directory and add your HTML templates inside it.

Example files:

```
templates/
├── welcome.html
└── reset-password.html
```

Example content of `welcome.html`:

```welcome.html
<!DOCTYPE html>
<html>
<head>
    <title>{{.subject}}</title>
</head>
<body>
    <h1>Welcome {{.name}}</h1>
</body>
</html>
```

## Attachments

Create a `attachments` directory in the root directory and add your attachments inside it.

Example files:

```
attachments/
├── document.pdf
└── local-file.jpg
```

## API Endpoints

### Send Email

```http
POST /api/v1/mails
```

Request body:

```json
{
	"to": "recipient@example.com",
	"subject": "Welcome {{.name}}",
	"template": "welcome", // template name without extension
	"data": {
		"name": "John Doe"
	},
	"attachments": [
		{
			"file": "local-file.jpg" // file name with extension
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
│   ├── config/ # Configuration and setup
│   ├── handlers/ # HTTP handlers
│   ├── middleware/ # HTTP middleware
│   ├── models/ # Database models
│   ├── routes/ # API routes
│   ├── services/ # Business logic
│   └── workers/ # Background workers
├── attachments/ # Email attachments
├── templates/ # Email templates
├── .env
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── main.go # Main application file
```

## Services

The application consists of the following services:

1. **API Service**: Main application service
2. **PostgreSQL**: Database service (v15)
3. **Redis**: Message broker and task queue (v7)
4. **Adminer**: Database management UI

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
