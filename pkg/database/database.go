package database

import (
	"fmt"
	"mailer-api/internal/models"
	"mailer-api/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupDatabase() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		config.Env.DB.Host, config.Env.DB.User, config.Env.DB.Pass, config.Env.DB.Name, config.Env.DB.Port,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// After database setup
	if err := db.AutoMigrate(&models.Mail{}, &models.Attachment{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}
