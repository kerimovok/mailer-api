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
	config  *QueueConfig
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

	// Get queue configuration
	queueConfig := DefaultQueueConfig()

	// Setup all queues and exchanges using shared configuration
	if err := queueConfig.SetupAllQueues(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to setup queues: %v", err)
	}

	// setupConnectionRecovery sets up automatic reconnection for RabbitMQ
	consumer := &Consumer{
		conn:    conn,
		channel: ch,
		config:  queueConfig,
	}
	consumer.setupConnectionRecovery()

	return consumer, nil
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
		c.config.QueueName, // queue
		"",                 // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
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

// setupConnectionRecovery sets up automatic reconnection for RabbitMQ
func (c *Consumer) setupConnectionRecovery() {
	// Monitor connection for errors
	go func() {
		for err := range c.conn.NotifyClose(make(chan *amqp.Error)) {
			if err != nil {
				log.Printf("RabbitMQ connection lost: %v, attempting to reconnect...", err)
				c.reconnect()
			}
		}
	}()

	// Monitor channel for errors
	go func() {
		for err := range c.channel.NotifyClose(make(chan *amqp.Error)) {
			if err != nil {
				log.Printf("RabbitMQ channel lost: %v, attempting to reconnect...", err)
				c.reconnect()
			}
		}
	}()
}

// reconnect attempts to reconnect to RabbitMQ
func (c *Consumer) reconnect() {
	for {
		log.Println("Attempting to reconnect to RabbitMQ...")

		// Close existing connections
		c.mu.Lock()
		if c.channel != nil {
			c.channel.Close()
		}
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()

		// Wait before retry
		time.Sleep(5 * time.Second)

		// Attempt to reconnect
		host := config.GetEnvOrDefault("RABBITMQ_HOST", "localhost")
		port := config.GetEnvOrDefault("RABBITMQ_PORT", "5672")
		username := config.GetEnvOrDefault("RABBITMQ_USERNAME", "guest")
		password := config.GetEnvOrDefault("RABBITMQ_PASSWORD", "guest")
		vhost := config.GetEnvOrDefault("RABBITMQ_VHOST", "/")

		url := fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
			username, password, host, port, vhost,
		)

		conn, err := amqp.Dial(url)
		if err != nil {
			log.Printf("Failed to reconnect: %v, retrying in 5 seconds...", err)
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			log.Printf("Failed to create channel: %v, retrying in 5 seconds...", err)
			conn.Close()
			continue
		}

		// Re-setup all queues and exchanges using shared configuration
		if err := c.config.SetupAllQueues(ch); err != nil {
			log.Printf("Failed to setup queues: %v, retrying in 5 seconds...", err)
			ch.Close()
			conn.Close()
			continue
		}

		// Update consumer with new connections
		c.mu.Lock()
		c.conn = conn
		c.channel = ch
		c.mu.Unlock()
		log.Println("Successfully reconnected to RabbitMQ")
		break
	}
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
			c.config.ExchangeName, // exchange
			c.config.RoutingKey,   // routing key
			false,                 // mandatory
			false,                 // immediate
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
