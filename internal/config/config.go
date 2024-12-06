package config

import (
	"mailer-api/pkg/utils"

	"github.com/joho/godotenv"
)

func LoadConfig() error {
	if err := godotenv.Load(); err != nil {
		if utils.GetEnv("GO_ENV") != "production" {
			return err
		}
	}

	return nil
}
