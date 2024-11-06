package config

import (
	"mailer-api/.internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, asynqClient *asynq.Client) {
	// Create handlers
	mailHandler := handlers.NewMailHandler(db, asynqClient)

	// API routes group
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Mail routes
	mail := v1.Group("/mails")
	mail.Post("/", mailHandler.SendMail)
	mail.Get("/", mailHandler.GetMails)
	mail.Get("/:id", mailHandler.GetMailByID)
}
