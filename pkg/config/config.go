package config

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"net/mail"
	"net/url"

	"github.com/joho/godotenv"
)

var AppConfig *Config

type Config struct {
	Server struct {
		Port        string
		Environment string
	}
	SMTP struct {
		Host     string
		Port     string
		Username string
		Password string
		From     string
	}
	DB struct {
		Host string
		Port string
		User string
		Pass string
		Name string
	}
	Redis struct {
		Addr     string
		Password string
	}
}

// ValidationRule defines a validation function that returns an error if validation fails
type ValidationRule struct {
	Field   string
	Rule    func(value string) bool
	Message string
}

func LoadEnv() error {
	if err := godotenv.Load(); err != nil {
		if os.Getenv("GO_ENV") != "production" {
			log.Printf("Warning: .env file not found")
		}
	}

	AppConfig = &Config{
		Server: struct {
			Port        string
			Environment string
		}{
			Port:        getEnvOrDefault("PORT", "3002"),
			Environment: getEnvOrDefault("GO_ENV", "development"),
		},
		SMTP: struct {
			Host     string
			Port     string
			Username string
			Password string
			From     string
		}{
			Host:     getEnvOrDefault("SMTP_HOST", ""),
			Port:     getEnvOrDefault("SMTP_PORT", ""),
			Username: getEnvOrDefault("SMTP_USERNAME", ""),
			Password: getEnvOrDefault("SMTP_PASSWORD", ""),
			From:     getEnvOrDefault("SMTP_FROM", ""),
		},
		DB: struct {
			Host string
			Port string
			User string
			Pass string
			Name string
		}{
			Host: getEnvOrDefault("DB_HOST", ""),
			Port: getEnvOrDefault("DB_PORT", ""),
			User: getEnvOrDefault("DB_USER", ""),
			Pass: getEnvOrDefault("DB_PASS", ""),
			Name: getEnvOrDefault("DB_NAME", ""),
		},
		Redis: struct {
			Addr     string
			Password string
		}{
			Addr:     getEnvOrDefault("REDIS_ADDR", ""),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
		},
	}

	return validateConfig()
}

// validateConfig checks all required configuration values
func validateConfig() error {
	rules := []ValidationRule{
		// Server validation
		{
			Field:   "Server.Port",
			Rule:    func(v string) bool { return v != "" },
			Message: "server port is required",
		},

		// SMTP validation
		{
			Field:   "SMTP.Host",
			Rule:    func(v string) bool { return v != "" },
			Message: "SMTP host is required",
		},
		{
			Field:   "SMTP.Port",
			Rule:    func(v string) bool { return v != "" },
			Message: "SMTP port is required",
		},
		{
			Field:   "SMTP.Username",
			Rule:    func(v string) bool { return v != "" },
			Message: "SMTP username is required",
		},
		{
			Field:   "SMTP.Password",
			Rule:    func(v string) bool { return v != "" },
			Message: "SMTP password is required",
		},

		// Database validation
		{
			Field:   "DB.Host",
			Rule:    func(v string) bool { return v != "" },
			Message: "database host is required",
		},
		{
			Field:   "DB.Port",
			Rule:    func(v string) bool { return v != "" },
			Message: "database port is required",
		},
		{
			Field:   "DB.User",
			Rule:    func(v string) bool { return v != "" },
			Message: "database user is required",
		},
		{
			Field:   "DB.Name",
			Rule:    func(v string) bool { return v != "" },
			Message: "database name is required",
		},

		// Redis validation (only if in production)
		{
			Field:   "Redis.Addr",
			Rule:    func(v string) bool { return AppConfig.Server.Environment != "production" || v != "" },
			Message: "Redis address is required in production",
		},
	}

	var errors []string
	for _, rule := range rules {
		value := getConfigValue(rule.Field)
		if !rule.Rule(value) {
			errors = append(errors, rule.Message)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// getConfigValue retrieves a configuration value using reflection based on the field path
func getConfigValue(fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	value := reflect.ValueOf(AppConfig).Elem()

	for _, part := range parts {
		value = value.FieldByName(part)
	}

	return value.String()
}

// AddValidationRule allows adding custom validation rules
func AddValidationRule(field string, rule func(string) bool, message string) {
	customRules = append(customRules, ValidationRule{
		Field:   field,
		Rule:    rule,
		Message: message,
	})
}

// Custom validation rules that can be added by the application
var customRules []ValidationRule

// Custom validation helper functions
func IsValidPort(port string) bool {
	if port == "" {
		return false
	}
	portNum, err := strconv.Atoi(port)
	return err == nil && portNum > 0 && portNum <= 65535
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func IsValidURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnv(key string) string {
	return os.Getenv(key)
}
