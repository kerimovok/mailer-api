package config

import (
	"github.com/joho/godotenv"
	"github.com/kerimovok/go-pkg-utils/config"
)

func LoadConfig() error {
	if err := godotenv.Load(); err != nil {
		if config.GetEnv("GO_ENV") != "production" {
			return nil
		}
	}

	return nil
}
