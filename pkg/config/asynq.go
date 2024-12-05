package config

import (
	"github.com/hibiken/asynq"
)

func SetupAsynq() (*asynq.Client, *asynq.Server) {
	// Redis connection setup
	redisConnection := asynq.RedisClientOpt{
		Addr:     AppConfig.Redis.Addr,
		Password: AppConfig.Redis.Password,
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
