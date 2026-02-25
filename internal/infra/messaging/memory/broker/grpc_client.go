package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"distributed-crawler/internal/infra/messaging"
	crawlergrpc "distributed-crawler/pkg/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcMessagingClient struct {
	conn   *grpc.ClientConn
	client crawlergrpc.MemoryBrokerServiceClient
}

// NewGRPCClient returns a messaging.Client that pushes and pulls messages
// from a remote memory_broker gRPC server at addr (host:port).
func NewGRPCClient(addr string) (messaging.Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to memory broker at %s: %w", addr, err)
	}
	return &grpcMessagingClient{
		conn:   conn,
		client: crawlergrpc.NewMemoryBrokerServiceClient(conn),
	}, nil
}

func (c *grpcMessagingClient) Publish(ctx context.Context, queueName string, message interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	_, err = c.client.Push(ctx, &crawlergrpc.MemoryPushRequest{Queue: queueName, Payload: payload})
	if err != nil {
		return fmt.Errorf("failed to push to queue %s: %w", queueName, err)
	}
	return nil
}

func (c *grpcMessagingClient) Consume(ctx context.Context, queueName string, handler func([]byte) error) error {
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.consumeLoop(ctx, queueName, handler); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("[memory-broker-client] error on queue %s, reconnecting in %v: %v", queueName, backoff, err)
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			backoff = time.Second
		}
	}
}

func (c *grpcMessagingClient) consumeLoop(ctx context.Context, queueName string, handler func([]byte) error) error {
	stream, err := c.client.Subscribe(ctx, &crawlergrpc.MemorySubscribeRequest{Queue: queueName})
	if err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		if err := handler(msg.Payload); err != nil {
			if messaging.IsNonRetryable(err) {
				log.Printf("[memory-broker-client] non-retryable error on queue %s, discarding: %v", queueName, err)
			} else {
				log.Printf("[memory-broker-client] retryable error on queue %s, requeuing: %v", queueName, err)
				reCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, pushErr := c.client.Push(reCtx, &crawlergrpc.MemoryPushRequest{Queue: queueName, Payload: msg.Payload})
				cancel()
				if pushErr != nil {
					log.Printf("[memory-broker-client] requeue failed for queue %s: %v", queueName, pushErr)
				}
			}
		}
	}
}

func (c *grpcMessagingClient) Close() error {
	return c.conn.Close()
}
