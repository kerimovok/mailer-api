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

var (
	Env *EnvConfig
)

type EnvConfig struct {
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

func LoadConfig() error {
	if err := godotenv.Load(); err != nil {
		if GetEnv("GO_ENV") != "production" {
			log.Printf("Warning: .env file not found")
		}
	}

	Env = &EnvConfig{
		Server: struct {
			Port        string
			Environment string
		}{
			Port:        GetEnvOrDefault("PORT", "3002"),
			Environment: GetEnvOrDefault("GO_ENV", "development"),
		},
		SMTP: struct {
			Host     string
			Port     string
			Username string
			Password string
			From     string
		}{
			Host:     GetEnvOrDefault("SMTP_HOST", ""),
			Port:     GetEnvOrDefault("SMTP_PORT", ""),
			Username: GetEnvOrDefault("SMTP_USERNAME", ""),
			Password: GetEnvOrDefault("SMTP_PASSWORD", ""),
			From:     GetEnvOrDefault("SMTP_FROM", ""),
		},
		DB: struct {
			Host string
			Port string
			User string
			Pass string
			Name string
		}{
			Host: GetEnvOrDefault("DB_HOST", ""),
			Port: GetEnvOrDefault("DB_PORT", ""),
			User: GetEnvOrDefault("DB_USER", ""),
			Pass: GetEnvOrDefault("DB_PASS", ""),
			Name: GetEnvOrDefault("DB_NAME", ""),
		},
		Redis: struct {
			Addr     string
			Password string
		}{
			Addr:     GetEnvOrDefault("REDIS_ADDR", ""),
			Password: GetEnvOrDefault("REDIS_PASSWORD", ""),
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
			Rule:    func(v string) bool { return Env.Server.Environment != "production" || v != "" },
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
	value := reflect.ValueOf(Env).Elem()

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

func GetEnvOrDefault(key, defaultValue string) string {
	if value := GetEnv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnv(key string) string {
	return os.Getenv(key)
}
