package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client interface {
	Publish(ctx context.Context, queueName string, message interface{}) error
	Consume(ctx context.Context, queueName string, handler func([]byte) error) error
	Close() error
}

type client struct {
	url  string
	conn *amqp.Connection
	mu   sync.Mutex
}

func NewClient(url string) (Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return &client{
		url:  url,
		conn: conn,
	}, nil
}

func (c *client) ensureConnection() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.conn.IsClosed() {
		return nil
	}

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
	}

	c.conn = conn
	return nil
}

func (c *client) Publish(ctx context.Context, queueName string, message interface{}) error {
	// Marshal message to JSON first (before connection attempts)
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Retry logic for transient failures
	maxRetries := 3
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Ensure connection is alive
		if err := c.ensureConnection(); err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to ensure connection after %d attempts: %w", maxRetries, err)
			}
			log.Printf("Failed to ensure connection (attempt %d/%d), retrying in %v: %v", attempt, maxRetries, backoff, err)
			select {
			case <-time.After(backoff):
				backoff *= 2
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Create a new channel for this publish operation
		ch, err := c.conn.Channel()
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to open channel after %d attempts: %w", maxRetries, err)
			}
			log.Printf("Failed to open channel (attempt %d/%d), retrying in %v: %v", attempt, maxRetries, backoff, err)
			select {
			case <-time.After(backoff):
				backoff *= 2
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Declare queue (idempotent - will create if doesn't exist)
		_, err = ch.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			ch.Close()
			if attempt == maxRetries {
				return fmt.Errorf("failed to declare queue after %d attempts: %w", maxRetries, err)
			}
			log.Printf("Failed to declare queue (attempt %d/%d), retrying in %v: %v", attempt, maxRetries, backoff, err)
			select {
			case <-time.After(backoff):
				backoff *= 2
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Publish message
		err = ch.PublishWithContext(
			ctx,
			"",        // exchange
			queueName, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         body,
			},
		)

		ch.Close()

		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to publish message after %d attempts: %w", maxRetries, err)
			}
			log.Printf("Failed to publish message (attempt %d/%d), retrying in %v: %v", attempt, maxRetries, backoff, err)
			select {
			case <-time.After(backoff):
				backoff *= 2
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Success
		return nil
	}

	return fmt.Errorf("failed to publish after %d attempts", maxRetries)
}

func (c *client) Consume(ctx context.Context, queueName string, handler func([]byte) error) error {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Ensure connection is alive
		if err := c.ensureConnection(); err != nil {
			log.Printf("Failed to ensure RabbitMQ connection, retrying in %v: %v", backoff, err)
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Reset backoff on successful connection
		backoff = time.Second

		// Create a new channel for consuming
		ch, err := c.conn.Channel()
		if err != nil {
			log.Printf("Failed to open RabbitMQ channel, retrying in %v: %v", backoff, err)
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Declare queue (idempotent - will create if doesn't exist)
		_, err = ch.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			ch.Close()
			return fmt.Errorf("failed to declare queue: %w", err)
		}

		// Set QoS - process one message at a time
		err = ch.Qos(
			1,     // prefetch count
			0,     // prefetch size
			false, // global
		)
		if err != nil {
			ch.Close()
			return fmt.Errorf("failed to set QoS: %w", err)
		}

		// Start consuming
		msgs, err := ch.Consume(
			queueName, // queue
			"",        // consumer tag
			false,     // auto-ack
			false,     // exclusive
			false,     // no-local
			false,     // no-wait
			nil,       // args
		)
		if err != nil {
			ch.Close()
			return fmt.Errorf("failed to start consuming: %w", err)
		}

		// Listen for channel close notifications
		closeChan := make(chan *amqp.Error, 1)
		ch.NotifyClose(closeChan)

		// Process messages with reconnection on channel/connection errors
		func() {
			defer ch.Close()

			for {
				select {
				case <-ctx.Done():
					return
				case err := <-closeChan:
					if err != nil {
						// Channel closed unexpectedly - will reconnect in outer loop
						return
					}
				case msg, ok := <-msgs:
					if !ok {
						// Message channel closed - will reconnect in outer loop
						return
					}

					// Process message
					if err := handler(msg.Body); err != nil {
						// Reject and requeue on error
						msg.Nack(false, true)
					} else {
						// Acknowledge on success
						msg.Ack(false)
					}
				}
			}
		}()

		// If we got here, the channel was closed - log and reconnect
		log.Printf("RabbitMQ channel closed for queue %s, reconnecting...", queueName)

		// Add a small delay before reconnecting
		select {
		case <-time.After(time.Second):
			// Continue to reconnect
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.conn.IsClosed() {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}
	return nil
}
