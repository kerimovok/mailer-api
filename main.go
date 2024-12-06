// main.go
package main

import (
	"mailer-api/internal/config"
	"mailer-api/internal/constants"
	"mailer-api/internal/routes"
	"mailer-api/internal/services"
	"mailer-api/internal/workers"
	"mailer-api/pkg/database"
	"mailer-api/pkg/utils"
	"mailer-api/pkg/validator"
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
	// Load all configs
	if err := config.LoadConfig(); err != nil {
		utils.LogFatal("failed to load configs", err)
	}

	// Validate environment variables
	if err := utils.ValidateConfig(constants.EnvValidationRules); err != nil {
		utils.LogFatal("configuration validation failed", err)
	}

	// Initialize validator
	validator.InitValidator()

	// Connect to database
	if err := database.ConnectDB(); err != nil {
		utils.LogFatal("failed to connect to database", err)
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
		utils.GetEnv("SMTP_HOST"),
		utils.GetEnv("SMTP_PORT"),
		utils.GetEnv("SMTP_USERNAME"),
		utils.GetEnv("SMTP_PASSWORD"),
		utils.GetEnv("SMTP_FROM"),
	)

	// Setup mail processor and workers
	mailProcessor := workers.NewMailProcessor(database.DB, mailService)
	if err := config.SetupWorkers(config.AsynqServer, mailProcessor); err != nil {
		utils.LogFatal("failed to setup workers", err)
	}

	// Setup routes
	routes.SetupRoutes(app)

	// Graceful shutdown channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		utils.LogInfo("shutting down server...")

		if err := app.Shutdown(); err != nil {
			utils.LogFatal("server forced to shutdown", err)
		}

		config.AsynqServer.Shutdown()
	}()

	// Start server
	utils.LogFatal("failed to start server", app.Listen(":"+utils.GetEnv("PORT")))

}
