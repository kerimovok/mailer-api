package config

import (
	"github.com/hibiken/asynq"
)

var AsynqClient *asynq.Client
var AsynqServer *asynq.Server

func ConnectAsynq() {
	redisConnection := asynq.RedisClientOpt{
		Addr:     Env.Redis.Addr,
		Password: Env.Redis.Password,
	}

	AsynqClient = asynq.NewClient(redisConnection)
	AsynqServer = asynq.NewServer(redisConnection, asynq.Config{
		Concurrency: 10,
	})
}
