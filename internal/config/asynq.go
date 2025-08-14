package config

import (
	"github.com/hibiken/asynq"
	"github.com/kerimovok/go-pkg-utils/config"
)

var AsynqClient *asynq.Client
var AsynqServer *asynq.Server

func ConnectAsynq() {
	redisConnection := asynq.RedisClientOpt{
		Addr:     config.GetEnv("REDIS_ADDR"),
		Password: config.GetEnv("REDIS_PASSWORD"),
	}

	AsynqClient = asynq.NewClient(redisConnection)
	AsynqServer = asynq.NewServer(redisConnection, asynq.Config{
		Concurrency: 10,
	})
}
