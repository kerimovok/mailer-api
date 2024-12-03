package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	SMTPFrom      string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPass        string
	DBName        string
	RedisAddr     string
	RedisPassword string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil && os.Getenv("GO_ENV") != "production" {
		return nil, err
	}

	return &Config{
		Port:          os.Getenv("PORT"),
		SMTPHost:      os.Getenv("SMTP_HOST"),
		SMTPPort:      os.Getenv("SMTP_PORT"),
		SMTPUsername:  os.Getenv("SMTP_USERNAME"),
		SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:      os.Getenv("SMTP_FROM"),
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        os.Getenv("DB_PORT"),
		DBUser:        os.Getenv("DB_USER"),
		DBPass:        os.Getenv("DB_PASS"),
		DBName:        os.Getenv("DB_NAME"),
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
	}, nil
}
