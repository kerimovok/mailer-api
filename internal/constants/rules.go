package constants

import (
	"github.com/kerimovok/go-pkg-utils/config"
	"github.com/kerimovok/go-pkg-utils/validator"
)

var EnvValidationRules = []validator.ValidationRule{
	// Server validation
	{
		Variable: "PORT",
		Default:  "3002",
		Rule:     config.IsValidPort,
		Message:  "server port is required and must be a valid port number",
	},
	{
		Variable: "GO_ENV",
		Default:  "development",
		Rule:     func(v string) bool { return v == "development" || v == "production" },
		Message:  "GO_ENV must be either 'development' or 'production'",
	},

	// Database validation
	{
		Variable: "DB_HOST",
		Rule:     config.IsValidNonEmptyString,
		Message:  "database host is required",
	},
	{
		Variable: "DB_PORT",
		Default:  "5432",
		Rule:     config.IsValidPort,
		Message:  "database port is required and must be a valid port number",
	},
	{
		Variable: "DB_USER",
		Rule:     config.IsValidNonEmptyString,
		Message:  "database user is required",
	},
	{
		Variable: "DB_PASS",
		Rule:     config.IsValidNonEmptyString,
		Message:  "database password is required",
	},
	{
		Variable: "DB_NAME",
		Default:  "auth",
		Rule:     config.IsValidNonEmptyString,
		Message:  "database name is required",
	},

	// SMTP validation
	{
		Variable: "SMTP_HOST",
		Rule:     config.IsValidNonEmptyString,
		Message:  "SMTP host is required",
	},
	{
		Variable: "SMTP_PORT",
		Default:  "587",
		Rule:     config.IsValidPort,
		Message:  "SMTP port is required and must be a valid port number",
	},
	{
		Variable: "SMTP_USERNAME",
		Rule:     config.IsValidNonEmptyString,
		Message:  "SMTP username is required",
	},
	{
		Variable: "SMTP_PASSWORD",
		Rule:     config.IsValidNonEmptyString,
		Message:  "SMTP password is required",
	},

	// SMTP From validation
	{
		Variable: "SMTP_FROM",
		Rule:     config.IsValidNonEmptyString,
		Message:  "SMTP from address is required",
	},

	// Email processing mode
	{
		Variable: "EMAIL_PROCESSING_MODE",
		Default:  "hybrid", // "rest-only", "queue-only", "hybrid"
		Rule: func(v string) bool {
			return v == "rest-only" || v == "queue-only" || v == "hybrid"
		},
		Message: "EMAIL_PROCESSING_MODE must be 'rest-only', 'queue-only', or 'hybrid'",
	},

	// Queue retry configuration
	{
		Variable: "QUEUE_MAX_RETRIES",
		Default:  "3",
		Rule:     config.IsValidInteger,
		Message:  "QUEUE_MAX_RETRIES must be a valid number",
	},
	{
		Variable: "QUEUE_RETRY_DELAY_BASE",
		Default:  "1",
		Rule:     config.IsValidInteger,
		Message:  "QUEUE_RETRY_DELAY_BASE must be a valid number (seconds)",
	},
	{
		Variable: "QUEUE_MAX_RETRY_DELAY",
		Default:  "300",
		Rule:     config.IsValidInteger,
		Message:  "QUEUE_MAX_RETRY_DELAY must be a valid number (seconds)",
	},

	// RabbitMQ validation (only required when EMAIL_PROCESSING_MODE includes queue processing)
	{
		Variable: "RABBITMQ_HOST",
		Default:  "localhost",
		Rule:     config.IsValidNonEmptyString,
		Message:  "RabbitMQ host is required when queue processing is enabled",
	},
	{
		Variable: "RABBITMQ_PORT",
		Default:  "5672",
		Rule:     config.IsValidPort,
		Message:  "RabbitMQ port must be a valid port number",
	},
	{
		Variable: "RABBITMQ_USERNAME",
		Default:  "guest",
		Rule:     config.IsValidNonEmptyString,
		Message:  "RabbitMQ username is required when queue processing is enabled",
	},
	{
		Variable: "RABBITMQ_PASSWORD",
		Default:  "guest",
		Rule:     config.IsValidNonEmptyString,
		Message:  "RabbitMQ password is required when queue processing is enabled",
	},
	{
		Variable: "RABBITMQ_VHOST",
		Default:  "/",
		Rule:     config.IsValidNonEmptyString,
		Message:  "RabbitMQ vhost is required when queue processing is enabled",
	},
}
