// main.go
package main

import (
	"log"
	"mailer-api/.internal/config"
	"mailer-api/.internal/services"
	"mailer-api/.internal/workers"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Setup database connection
	db, err := config.SetupDatabase(cfg)
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

	// Setup Fiber app
	app := fiber.New()

	// Setup routes
	config.SetupRoutes(app, db, asynqClient)

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
