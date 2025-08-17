package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"mailer-api/internal/services"
	"strconv"
	"sync"
	"time"

	"github.com/kerimovok/go-pkg-utils/config"
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
	host := config.GetEnvOrDefault("RABBITMQ_HOST", "localhost")
	port := config.GetEnvOrDefault("RABBITMQ_PORT", "5672")
	username := config.GetEnvOrDefault("RABBITMQ_USERNAME", "guest")
	password := config.GetEnvOrDefault("RABBITMQ_PASSWORD", "guest")
	vhost := config.GetEnvOrDefault("RABBITMQ_VHOST", "/")

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

	// Setup dead letter queue first
	if err := setupDeadLetterQueue(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to setup dead letter queue: %v", err)
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

	// Declare queue with dead letter configuration
	q, err := ch.QueueDeclare(
		"email_queue", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		amqp.Table{ // arguments for additional durability and dead letter handling
			"x-message-ttl":             int32(24 * 60 * 60 * 1000), // 24 hours TTL
			"x-max-priority":            int32(10),                  // Priority support
			"x-overflow":                "drop-head",                // Drop oldest when full
			"x-dead-letter-exchange":    "mailer.dlx",               // Dead letter exchange
			"x-dead-letter-routing-key": "email.failed",             // Routing key for DLQ
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

// setupDeadLetterQueue sets up the dead letter exchange and queue for failed messages
func setupDeadLetterQueue(ch *amqp.Channel) error {
	// Declare dead letter exchange
	err := ch.ExchangeDeclare(
		"mailer.dlx", // dead letter exchange name
		"direct",     // exchange type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %v", err)
	}

	// Declare dead letter queue
	dlq, err := ch.QueueDeclare(
		"email_dlq", // dead letter queue name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %v", err)
	}

	// Bind dead letter queue to exchange
	err = ch.QueueBind(
		dlq.Name,       // queue name
		"email.failed", // routing key
		"mailer.dlx",   // exchange
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %v", err)
	}

	return nil
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
	// Get retry count from message headers
	retryCount := getRetryCount(msg)
	maxRetries := getMaxRetries()

	if retryCount >= maxRetries {
		// Max retries exceeded, reject message (will go to DLQ)
		log.Printf("Max retries exceeded for email task, message will go to DLQ")
		if err := msg.Reject(false); err != nil {
			log.Printf("Failed to reject message after max retries: %v", err)
		}
		return
	}

	var emailTask EmailTask
	if err := json.Unmarshal(msg.Body, &emailTask); err != nil {
		log.Printf("Failed to unmarshal email task: %v", err)
		// Bad message format, reject and send to DLQ
		if err := msg.Reject(false); err != nil {
			log.Printf("Failed to reject malformed message: %v", err)
		}
		return
	}

	log.Printf("Processing email task for user: %s, type: %s (attempt %d/%d)", emailTask.To, emailTask.Type, retryCount+1, maxRetries)

	// Use unified email processing (note: queue-based emails typically don't have attachments)
	mail, err := services.ProcessEmailRequest(emailTask.To, emailTask.Subject, emailTask.Template, emailTask.Data, nil)
	if err != nil {
		log.Printf("Failed to process email task (attempt %d/%d): %v", retryCount+1, maxRetries, err)

		// Increment retry count and requeue with delay
		newHeaders := amqp.Table{}
		if msg.Headers == nil {
			msg.Headers = amqp.Table{}
		}
		newHeaders["x-retry-count"] = retryCount + 1
		newHeaders["x-last-error"] = err.Error()
		newHeaders["x-last-retry"] = time.Now().Unix()

		// Reject and requeue with delay
		if err := msg.Reject(false); err != nil {
			log.Printf("Failed to reject message for retry: %v", err)
		}

		// Schedule retry with exponential backoff
		c.scheduleRetry(msg.Body, newHeaders, calculateRetryDelay(retryCount))
		return
	}

	// Success - acknowledge message
	if err := msg.Ack(false); err != nil {
		log.Printf("Failed to acknowledge message: %v", err)
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

// getMaxRetries gets the maximum number of retries from configuration
func getMaxRetries() int {
	maxRetries := config.GetEnv("QUEUE_MAX_RETRIES")
	if maxRetries == "" {
		maxRetries = "3" // default value
	}

	count, err := strconv.Atoi(maxRetries)
	if err != nil {
		log.Printf("Invalid QUEUE_MAX_RETRIES value '%s', using default: 3", maxRetries)
		return 3
	}

	return count
}

// getRetryDelayBase gets the base delay for retries from configuration
func getRetryDelayBase() int {
	baseDelay := config.GetEnv("QUEUE_RETRY_DELAY_BASE")
	if baseDelay == "" {
		baseDelay = "1" // default value
	}

	delay, err := strconv.Atoi(baseDelay)
	if err != nil {
		log.Printf("Invalid QUEUE_RETRY_DELAY_BASE value '%s', using default: 1", baseDelay)
		return 1
	}

	return delay
}

// getMaxRetryDelay gets the maximum retry delay from configuration
func getMaxRetryDelay() int {
	maxDelay := config.GetEnv("QUEUE_MAX_RETRY_DELAY")
	if maxDelay == "" {
		maxDelay = "300" // default value (5 minutes)
	}

	delay, err := strconv.Atoi(maxDelay)
	if err != nil {
		log.Printf("Invalid QUEUE_MAX_RETRY_DELAY value '%s', using default: 300", maxDelay)
		return 300
	}

	return delay
}

// getRetryCount extracts the retry count from message headers
func getRetryCount(msg amqp.Delivery) int {
	if msg.Headers != nil {
		if retryCount, exists := msg.Headers["x-retry-count"]; exists {
			if count, ok := retryCount.(int32); ok {
				return int(count)
			}
		}
	}
	return 0
}

// calculateRetryDelay calculates delay with exponential backoff
func calculateRetryDelay(retryCount int) time.Duration {
	// Get configuration values
	baseDelay := getRetryDelayBase()
	maxDelay := getMaxRetryDelay()

	// Exponential backoff: baseDelay * 2^retryCount
	delay := time.Duration(baseDelay) * time.Duration(1<<retryCount) * time.Second

	// Cap at max delay
	if delay > time.Duration(maxDelay)*time.Second {
		delay = time.Duration(maxDelay) * time.Second
	}

	return delay
}

// scheduleRetry schedules a message for retry with delay
func (c *Consumer) scheduleRetry(body []byte, headers amqp.Table, delay time.Duration) {
	// In a production system, you might want to use a proper delay queue
	// For now, we'll use a simple goroutine with sleep
	go func() {
		time.Sleep(delay)

		// Publish back to the main queue with updated headers
		err := c.channel.Publish(
			"mailer", // exchange
			"email",  // routing key
			false,    // mandatory
			false,    // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         body,
				Headers:      headers,
				DeliveryMode: amqp.Persistent,
			},
		)

		if err != nil {
			log.Printf("Failed to schedule retry: %v", err)
		} else {
			log.Printf("Scheduled retry with delay %v", delay)
		}
	}()
}
