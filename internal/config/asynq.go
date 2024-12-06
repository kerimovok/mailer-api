package config

import (
	"mailer-api/pkg/utils"

	"github.com/hibiken/asynq"
)

var AsynqClient *asynq.Client
var AsynqServer *asynq.Server

func ConnectAsynq() {
	redisConnection := asynq.RedisClientOpt{
		Addr:     utils.GetEnv("REDIS_ADDR"),
		Password: utils.GetEnv("REDIS_PASSWORD"),
	}

	AsynqClient = asynq.NewClient(redisConnection)
	AsynqServer = asynq.NewServer(redisConnection, asynq.Config{
		Concurrency: 10,
	})
}
