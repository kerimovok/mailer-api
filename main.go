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
	// Get service configuration
	emailProcessingMode := pkgConfig.GetEnv("EMAIL_PROCESSING_MODE")
	enableRestAPI := emailProcessingMode == "rest-only" || emailProcessingMode == "hybrid"
	enableRabbitMQConsumer := emailProcessingMode == "queue-only" || emailProcessingMode == "hybrid"

	log.Printf("Email processing mode: %s", emailProcessingMode)
	log.Printf("Service configuration: REST API=%v, RabbitMQ Consumer=%v", enableRestAPI, enableRabbitMQConsumer)

	// Validate processing mode
	if emailProcessingMode != "rest-only" && emailProcessingMode != "queue-only" && emailProcessingMode != "hybrid" {
		log.Fatal("Invalid EMAIL_PROCESSING_MODE. Must be 'rest-only', 'queue-only', or 'hybrid'")
	}

	var app *fiber.App
	var consumer *queue.Consumer

	// Setup Fiber app only if REST API is enabled
	if enableRestAPI {
		app = setupApp()
		// Setup routes
		routes.SetupRoutes(app)
		log.Println("REST API server initialized")
	}

	// Setup RabbitMQ consumer only if enabled
	if enableRabbitMQConsumer {
		var err error
		consumer, err = queue.NewConsumer()
		if err != nil {
			log.Printf("Failed to initialize RabbitMQ consumer: %v", err)
			log.Println("Continuing without RabbitMQ consumer...")
			enableRabbitMQConsumer = false
		} else {
			// Start consuming messages in background
			go func() {
				if err := consumer.StartConsuming(); err != nil {
					log.Printf("RabbitMQ consumer error: %v", err)
				}
			}()
			log.Println("RabbitMQ consumer initialized")
		}
	}

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Gracefully shutting down...")

		// Shutdown the server if REST API is enabled
		if enableRestAPI && app != nil {
			if err := app.Shutdown(); err != nil {
				log.Printf("error during server shutdown: %v", err)
			}
		}

		// Close RabbitMQ consumer if enabled
		if enableRabbitMQConsumer && consumer != nil {
			if err := consumer.Close(); err != nil {
				log.Printf("error during consumer shutdown: %v", err)
			}
		}

		log.Println("Server gracefully stopped")
		os.Exit(0)
	}()

	// Start server only if REST API is enabled
	if enableRestAPI {
		log.Printf("Starting REST API server on port %s", pkgConfig.GetEnv("PORT"))
		if err := app.Listen(":" + pkgConfig.GetEnv("PORT")); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	} else {
		log.Println("REST API is disabled, running in consumer-only mode")
		// Keep the main goroutine alive for the consumer
		select {}
	}
}
