// memory_broker is a lightweight gRPC server that replaces RabbitMQ / Kafka
// in development and test environments. It stores messages in buffered Go
// channels — all messages are lost on restart (intentional).
//
// Usage:
//
//	memory_broker --addr :9090 --capacity 1000
//
// Or via environment variables:
//
//	MEMORY_BROKER_ADDR=:9090 MEMORY_BROKER_CAPACITY=1000 memory_broker
//
// Workers connect to this server by setting:
//
//	MESSAGING_BROKER=grpc_memory
//	MEMORY_BROKER_ADDR=<host>:<port>
package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"distributed-crawler/internal/infra/messaging/memory/broker"
	crawlergrpc "distributed-crawler/pkg/v1"

	"google.golang.org/grpc"
)

func main() {
	addr := flag.String("addr", ":9095", "gRPC listen address")
	capacity := flag.Int("capacity", 1000, "per-queue message buffer capacity")
	flag.Parse()

	if v := os.Getenv("MEMORY_BROKER_ADDR"); v != "" {
		*addr = v
	}
	if v := os.Getenv("MEMORY_BROKER_CAPACITY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			*capacity = n
		}
	}

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *addr, err)
	}

	srv := grpc.NewServer()
	crawlergrpc.RegisterMemoryBrokerServiceServer(srv, broker.NewBrokerServer(*capacity))

	log.Printf("memory-broker: listening on %s (queue capacity per topic: %d)", *addr, *capacity)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("memory-broker: shutting down gracefully…")
		srv.GracefulStop()
	}()

	if err := srv.Serve(lis); err != nil {
		log.Fatalf("memory-broker: serve error: %v", err)
	}
}
