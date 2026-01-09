package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client interface {
	Publish(ctx context.Context, queueName string, message interface{}) error
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
	// Ensure connection is alive
	if err := c.ensureConnection(); err != nil {
		return err
	}

	// Create a new channel for this publish operation
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

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
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Marshal message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
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
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
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
