package routes

import (
	"mailer-api/internal/controllers"

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
	mail.Post("/", controllers.SendMail)
	mail.Get("/", controllers.GetMails)
	mail.Get("/:id", controllers.GetMailByID)

	// TODO: Add routes for attachments
}
