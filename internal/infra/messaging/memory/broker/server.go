package broker

import (
	"context"
	"fmt"
	"log"
	"sync"

	crawlergrpc "distributed-crawler/pkg/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultCapacity = 1000

// BrokerServer is the gRPC implementation of crawlergrpc.MemoryBrokerServiceServer.
// Queues are backed by buffered channels — all messages are lost on restart.
type BrokerServer struct {
	crawlergrpc.UnimplementedMemoryBrokerServiceServer

	mu       sync.Mutex
	queues   map[string]chan []byte
	queueCap int
}

// NewBrokerServer creates a new in-memory broker server.
// capacity ≤ 0 uses defaultCapacity.
func NewBrokerServer(capacity int) *BrokerServer {
	if capacity <= 0 {
		capacity = defaultCapacity
	}
	return &BrokerServer{
		queues:   make(map[string]chan []byte),
		queueCap: capacity,
	}
}

func (s *BrokerServer) queue(name string) chan []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	if q, ok := s.queues[name]; ok {
		return q
	}
	q := make(chan []byte, s.queueCap)
	s.queues[name] = q
	return q
}

func (s *BrokerServer) Push(_ context.Context, req *crawlergrpc.MemoryPushRequest) (*crawlergrpc.MemoryPushResponse, error) {
	if req.Queue == "" {
		return nil, status.Errorf(codes.InvalidArgument, "queue name is required")
	}
	q := s.queue(req.Queue)
	select {
	case q <- req.Payload:
		log.Printf("[memory-broker] push to queue %q (%d bytes, depth %d/%d)", req.Queue, len(req.Payload), len(q), s.queueCap)
		return &crawlergrpc.MemoryPushResponse{}, nil
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "queue %q is full (capacity %d)", req.Queue, s.queueCap)
	}
}

func (s *BrokerServer) Subscribe(req *crawlergrpc.MemorySubscribeRequest, stream grpc.ServerStreamingServer[crawlergrpc.MemoryBrokerMessage]) error {
	if req.Queue == "" {
		return status.Errorf(codes.InvalidArgument, "queue name is required")
	}
	q := s.queue(req.Queue)
	log.Printf("[memory-broker] subscriber connected to queue %q", req.Queue)
	defer log.Printf("[memory-broker] subscriber disconnected from queue %q", req.Queue)

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case payload := <-q:
			log.Printf("[memory-broker] sending to subscriber on queue %q (%d bytes)", req.Queue, len(payload))
			if err := stream.Send(&crawlergrpc.MemoryBrokerMessage{Payload: payload}); err != nil {
				select {
				case q <- payload:
				default:
					log.Printf("[memory-broker] queue %q full, dropping message on requeue", req.Queue)
				}
				return fmt.Errorf("send failed: %w", err)
			}
		}
	}
}
