package main

import (
	"log"
	"mailer-api/internal/config"
	"mailer-api/internal/constants"
	"mailer-api/internal/database"
	"mailer-api/internal/queue"
	"mailer-api/internal/routes"
	"mailer-api/internal/services"
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
	pkgConfig "github.com/kerimovok/go-pkg-utils/config"
	pkgValidator "github.com/kerimovok/go-pkg-utils/validator"
)

func init() {
	// Load all configs
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("failed to load configs: %v", err)
	}

	// Validate environment variables
	if err := pkgValidator.ValidateConfig(constants.EnvValidationRules); err != nil {
		log.Fatalf("configuration validation failed: %v", err)
	}

	// Connect to database
	if err := database.ConnectDB(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Initialize services
	services.InitMailService()
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

	// Setup RabbitMQ consumer
	consumer := queue.NewConsumer()
	defer consumer.Close()

	// Start consuming messages in background
	go consumer.StartConsuming()

	// Setup routes
	routes.SetupRoutes(app)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Gracefully shutting down...")

		// Shutdown the server
		if err := app.Shutdown(); err != nil {
			log.Printf("error during server shutdown: %v", err)
		}

		// Close RabbitMQ consumer
		consumer.Close()

		log.Println("Server gracefully stopped")
		os.Exit(0)
	}()

	// Start server
	if err := app.Listen(":" + pkgConfig.GetEnv("PORT")); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server: %v", err)
	}
}
