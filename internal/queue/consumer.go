package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"mailer-api/internal/database"
	"mailer-api/internal/models"
	"mailer-api/internal/services"
	"os"
	"sync"

	"github.com/kerimovok/go-pkg-database/sql"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.RWMutex // Protect connection updates
}

type EmailTask struct {
	To       string                 `json:"to"`
	Subject  string                 `json:"subject"`
	Template string                 `json:"template"`
	Data     map[string]interface{} `json:"data"`
	Type     string                 `json:"type"`
}

func NewConsumer() *Consumer {
	// Get RabbitMQ connection details from environment variables
	host := getEnvOrDefault("RABBITMQ_HOST", "localhost")
	port := getEnvOrDefault("RABBITMQ_PORT", "5672")
	username := getEnvOrDefault("RABBITMQ_USERNAME", "guest")
	password := getEnvOrDefault("RABBITMQ_PASSWORD", "guest")
	vhost := getEnvOrDefault("RABBITMQ_VHOST", "/")

	// Connect to RabbitMQ
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		username,
		password,
		host,
		port,
		vhost,
	)

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}

	// Declare exchange
	err = ch.ExchangeDeclare(
		"mailer", // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	// Declare queue
	q, err := ch.QueueDeclare(
		"email_queue", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		amqp.Table{ // arguments for additional durability
			"x-message-ttl":  int32(24 * 60 * 60 * 1000), // 24 hours TTL
			"x-max-priority": int32(10),                  // Priority support
			"x-overflow":     "drop-head",                // Drop oldest when full
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,   // queue name
		"email",  // routing key
		"mailer", // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	// Set QoS
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		log.Fatalf("Failed to set QoS: %v", err)
	}

	return &Consumer{
		conn:    conn,
		channel: ch,
	}
}

func (c *Consumer) StartConsuming() {
	// Check connection health before starting
	c.mu.RLock()
	if c.conn == nil || c.conn.IsClosed() || c.channel == nil || c.channel.IsClosed() {
		c.mu.RUnlock()
		log.Fatal("RabbitMQ connection is not available")
	}
	c.mu.RUnlock()

	msgs, err := c.channel.Consume(
		"email_queue", // queue
		"",            // consumer
		false,         // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	log.Println("Starting to consume email tasks from RabbitMQ...")

	for msg := range msgs {
		go c.processEmailTask(msg)
	}
}

// IsConnected returns true if the consumer has a valid connection
func (c *Consumer) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && !c.conn.IsClosed() && c.channel != nil && !c.channel.IsClosed()
}

func (c *Consumer) processEmailTask(msg amqp.Delivery) {
	defer func() {
		if err := msg.Ack(false); err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		}
	}()

	var emailTask EmailTask
	if err := json.Unmarshal(msg.Body, &emailTask); err != nil {
		log.Printf("Failed to unmarshal email task: %v", err)
		return
	}

	log.Printf("Processing email task for user: %s, type: %s", emailTask.To, emailTask.Type)

	// Create mail record
	mail := models.Mail{
		To:       emailTask.To,
		Subject:  emailTask.Subject,
		Template: emailTask.Template,
		Data:     sql.JSONB(emailTask.Data),
		Status:   "pending",
	}

	// Use WithTransaction helper
	err := sql.WithTransaction(database.DB, func(tx *gorm.DB) error {
		// Create mail record
		if err := tx.Create(&mail).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to create mail record: %v", err)
		return
	}

	// Convert data to string for SendMail
	dataStr, err := json.Marshal(emailTask.Data)
	if err != nil {
		log.Printf("Failed to marshal email data: %v", err)
		mail.Status = "failed"
		mail.Error = err.Error()
		if saveErr := database.DB.Save(&mail).Error; saveErr != nil {
			log.Printf("Failed to save failed mail status: %v", saveErr)
		}
		return
	}

	// Send the email
	err = services.SendMail(emailTask.To, emailTask.Subject, emailTask.Template, string(dataStr), nil)
	if err != nil {
		log.Printf("Failed to send email: %v", err)
		mail.Status = "failed"
		mail.Error = err.Error()
	} else {
		log.Printf("Email sent successfully to: %s", emailTask.To)
		mail.Status = "sent"
	}

	// Update mail status
	if err := database.DB.Save(&mail).Error; err != nil {
		log.Printf("Failed to update mail status: %v", err)
		// TODO: Add to a retry queue or dead letter queue
		return
	}
}

func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.channel.Close(); err != nil {
		return err
	}
	return c.conn.Close()
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
