package messaging

import (
	"context"
	"errors"
)

// Client is the common interface for message broker clients (RabbitMQ, Kafka, etc.)
type Client interface {
	Publish(ctx context.Context, queueName string, message interface{}) error
	Consume(ctx context.Context, queueName string, handler func([]byte) error) error
	Close() error
}

// NonRetryableError wraps an error that should not cause the message to be requeued.
// The message will be discarded / sent to DLQ.
type NonRetryableError struct {
	Err error
}

func (e *NonRetryableError) Error() string { return e.Err.Error() }
func (e *NonRetryableError) Unwrap() error { return e.Err }

// NewNonRetryableError wraps err so that Consume discards the message instead of requeuing.
func NewNonRetryableError(err error) error { return &NonRetryableError{Err: err} }

// IsNonRetryable reports whether err (or any in its chain) is a NonRetryableError.
func IsNonRetryable(err error) bool {
	var nre *NonRetryableError
	return errors.As(err, &nre)
}
