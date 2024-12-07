package routes

import (
	"mailer-api/internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

func SetupRoutes(app *fiber.App) {
	// API routes group
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Monitor route
	app.Get("/metrics", monitor.New())

	// Mail routes
	mail := v1.Group("/mails")
	mail.Post("/", handlers.SendMail)
	mail.Get("/", handlers.GetMails)
	mail.Get("/:id", handlers.GetMailByID)

	// TODO: Add routes for attachments
}
