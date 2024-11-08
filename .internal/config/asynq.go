package config

import (
	"os"

	"github.com/hibiken/asynq"
)

func SetupAsynq(cfg *Config) (*asynq.Client, *asynq.Server) {
	// Redis connection setup
	redisConnection := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	}
	// Asynq client setup
	asynqClient := asynq.NewClient(redisConnection)

	// Asynq server setup
	asynqServer := asynq.NewServer(
		redisConnection,
		asynq.Config{
			Concurrency: 10,
		},
	)

	return asynqClient, asynqServer
}
