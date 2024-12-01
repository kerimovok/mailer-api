package config

import (
	"log"
	"mailer-api/internal/workers"
	"mailer-api/pkg/constants"

	"github.com/hibiken/asynq"
)

func SetupWorkers(server *asynq.Server, processor *workers.MailProcessor) error {
	mux := asynq.NewServeMux()

	// Register task controllers
	mux.HandleFunc(constants.TaskTypeSendEmail, processor.ProcessMail)

	// Start Asynq server
	go func() {
		if err := server.Run(mux); err != nil {
			log.Fatal(err)
		}
	}()

	return nil
}
