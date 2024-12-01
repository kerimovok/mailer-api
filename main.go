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
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Setup Fiber app
	app := setupApp()

	// Setup database connection
	db, err := database.SetupDatabase(cfg)
	if err != nil {
		log.Fatal("Failed to setup database:", err)
	}

	// Setup Redis and Asynq
	asynqClient, server := config.SetupAsynq(cfg)
	defer asynqClient.Close()

	// Setup mail service
	mailService := services.NewMailService(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUsername,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
	)

	// Setup mail processor and workers
	mailProcessor := workers.NewMailProcessor(db, mailService)
	if err := config.SetupWorkers(server, mailProcessor); err != nil {
		log.Fatal("Failed to setup workers:", err)
	}

	// Setup routes
	routes.SetupRoutes(app, db, asynqClient)

	// Graceful shutdown channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")

		if err := app.Shutdown(); err != nil {
			log.Fatal("Server forced to shutdown:", err)
		}

		server.Shutdown()
	}()

	// Start server
	if err := app.Listen(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
		log.Fatal("Failed to start server:", err)
	}
}
