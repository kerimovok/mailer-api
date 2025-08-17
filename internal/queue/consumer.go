package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"mailer-api/internal/services"
	"os"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
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

func NewConsumer() (*Consumer, error) {
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
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %v", err)
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
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %v", err)
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
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %v", err)
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
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %v", err)
	}

	return &Consumer{
		conn:    conn,
		channel: ch,
	}, nil
}

func (c *Consumer) StartConsuming() error {
	// Check connection health before starting
	c.mu.RLock()
	if c.conn == nil || c.conn.IsClosed() || c.channel == nil || c.channel.IsClosed() {
		c.mu.RUnlock()
		return fmt.Errorf("RabbitMQ connection is not available")
	}
	c.mu.RUnlock()

	// Set QoS for better message handling
	err := c.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

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
		return fmt.Errorf("failed to register a consumer: %v", err)
	}

	log.Println("Starting to consume email tasks from RabbitMQ...")

	for msg := range msgs {
		go c.processEmailTask(msg)
	}

	return nil
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

	// Use unified email processing (note: queue-based emails typically don't have attachments)
	mail, err := services.ProcessEmailRequest(emailTask.To, emailTask.Subject, emailTask.Template, emailTask.Data, nil)
	if err != nil {
		log.Printf("Failed to process email task: %v", err)
		// The unified method already handles status updates, so we don't need to do anything here
		return
	}

	log.Printf("Email processed successfully from queue: %s", mail.ID.String())
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
