package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"distributed-crawler/internal/infra/messaging"

	kafkago "github.com/segmentio/kafka-go"
)

type client struct {
	brokers       []string
	consumerGroup string

	writersMu sync.Mutex
	writers   map[string]*kafkago.Writer
}

// NewClient creates a new Kafka messaging client.
func NewClient(brokers []string, consumerGroup string) (messaging.Client, error) {
	return &client{
		brokers:       brokers,
		consumerGroup: consumerGroup,
		writers:       make(map[string]*kafkago.Writer),
	}, nil
}

func (c *client) getWriter(topic string) *kafkago.Writer {
	c.writersMu.Lock()
	defer c.writersMu.Unlock()

	if w, ok := c.writers[topic]; ok {
		return w
	}

	w := &kafkago.Writer{
		Addr:                   kafkago.TCP(c.brokers...),
		Topic:                  topic,
		Balancer:               &kafkago.LeastBytes{},
		WriteTimeout:           10 * time.Second,
		RequiredAcks:           kafkago.RequireOne,
		AllowAutoTopicCreation: true,
		MaxAttempts:            3,
	}
	c.writers[topic] = w
	return w
}

func (c *client) Publish(ctx context.Context, topic string, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	w := c.getWriter(topic)
	if err := w.WriteMessages(ctx, kafkago.Message{Value: body}); err != nil {
		return fmt.Errorf("failed to publish to kafka topic %s: %w", topic, err)
	}
	return nil
}

func (c *client) Consume(ctx context.Context, topic string, handler func([]byte) error) error {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		r := kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:     c.brokers,
			Topic:       topic,
			GroupID:     c.consumerGroup,
			MinBytes:    1,
			MaxBytes:    10e6, // 10 MB
			StartOffset: kafkago.FirstOffset,
		})

		err := c.consumeLoop(ctx, r, topic, handler)
		r.Close()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			log.Printf("Kafka consumer loop error for topic %s, reconnecting in %v: %v", topic, backoff, err)
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func (c *client) consumeLoop(ctx context.Context, r *kafkago.Reader, topic string, handler func([]byte) error) error {
	retryBackoff := time.Second
	maxRetryBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("failed to fetch message from topic %s: %w", topic, err)
		}

		retryBackoff = time.Second

		if err := handler(msg.Value); err != nil {
			if messaging.IsNonRetryable(err) {
				log.Printf("Non-retryable error processing message from topic %s, discarding: %v", topic, err)
				if commitErr := r.CommitMessages(ctx, msg); commitErr != nil && ctx.Err() == nil {
					log.Printf("Failed to commit discarded message from topic %s: %v", topic, commitErr)
				}
			} else {
				// Retryable — do not commit; message will be re-delivered after consumer restart/rebalance.
				log.Printf("Error processing message from topic %s, will retry: %v", topic, err)
				select {
				case <-time.After(retryBackoff):
					retryBackoff = min(retryBackoff*2, maxRetryBackoff)
				case <-ctx.Done():
					return nil
				}
			}
		} else {
			if err := r.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil {
				log.Printf("Failed to commit message from topic %s: %v", topic, err)
			}
		}
	}
}

func (c *client) Close() error {
	c.writersMu.Lock()
	defer c.writersMu.Unlock()

	for _, w := range c.writers {
		w.Close()
	}
	return nil
}
