package constants

import "mailer-api/pkg/utils"

var EnvValidationRules = []utils.ValidationRule{
	// Server validation
	{
		Variable: "PORT",
		Default:  "3002",
		Rule:     utils.IsValidPort,
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
		Rule:     func(v string) bool { return v != "" },
		Message:  "database host is required",
	},
	{
		Variable: "DB_PORT",
		Default:  "5432",
		Rule:     utils.IsValidPort,
		Message:  "database port is required and must be a valid port number",
	},
	{
		Variable: "DB_USER",
		Rule:     func(v string) bool { return v != "" },
		Message:  "database user is required",
	},
	{
		Variable: "DB_NAME",
		Default:  "auth",
		Rule:     func(v string) bool { return v != "" },
		Message:  "database name is required",
	},

	// SMTP validation
	{
		Variable: "SMTP_HOST",
		Rule:     func(v string) bool { return v != "" },
		Message:  "SMTP host is required",
	},
	{
		Variable: "SMTP_PORT",
		Default:  "587",
		Rule:     utils.IsValidPort,
		Message:  "SMTP port is required and must be a valid port number",
	},
	{
		Variable: "SMTP_USERNAME",
		Rule:     func(v string) bool { return v != "" },
		Message:  "SMTP username is required",
	},
	{
		Variable: "SMTP_PASSWORD",
		Rule:     func(v string) bool { return v != "" },
		Message:  "SMTP password is required",
	},

	// Redis validation
	{
		Variable: "REDIS_ADDR",
		Rule:     func(v string) bool { return v != "" },
		Message:  "Redis address is required",
	},
}
