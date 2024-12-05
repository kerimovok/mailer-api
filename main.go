// main.go
package main

import (
	"log"
	"mailer-api/internal/routes"
	"mailer-api/internal/services"
	"mailer-api/internal/workers"
	"mailer-api/pkg/config"
	"mailer-api/pkg/database"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

func init() {
	// Load environment variables
	if err := config.LoadConfig(); err != nil {
		log.Fatal(err)
	}

	// Connect to database
	if err := database.ConnectDB(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}
}

func setupApp() *fiber.App {
	app := fiber.New(fiber.Config{})

	// Middleware
	app.Use(helmet.New())
	app.Use(cors.New())
	app.Use(compress.New())
	app.Use(healthcheck.New())
	app.Use(requestid.New(requestid.Config{
		Generator: func() string {
			return uuid.New().String()
		},
	}))
	app.Use(logger.New())

	return app
}

func main() {
	// Setup Fiber app
	app := setupApp()

	// Setup Redis and Asynq
	config.ConnectAsynq()
	defer config.AsynqClient.Close()

	// Setup mail service
	mailService := services.NewMailService(
		config.Env.SMTP.Host,
		config.Env.SMTP.Port,
		config.Env.SMTP.Username,
		config.Env.SMTP.Password,
		config.Env.SMTP.From,
	)

	// Setup mail processor and workers
	mailProcessor := workers.NewMailProcessor(database.DB, mailService)
	if err := config.SetupWorkers(config.AsynqServer, mailProcessor); err != nil {
		log.Fatal("Failed to setup workers:", err)
	}

	// Setup routes
	routes.SetupRoutes(app)

	// Graceful shutdown channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")

		if err := app.Shutdown(); err != nil {
			log.Fatal("Server forced to shutdown:", err)
		}

		config.AsynqServer.Shutdown()
	}()

	// Start server
	if err := app.Listen(":" + config.Env.Server.Port); err != nil && err != http.ErrServerClosed {
		log.Fatal("Failed to start server:", err)
	}
}
