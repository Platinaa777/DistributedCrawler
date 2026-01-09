package main

import (
	"context"
	"distributed-crawler/internal/app"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()

	workerApp, err := app.NewWorkerApp(ctx, app.FetchWorkerType)
	if err != nil {
		log.Fatalf("failed to init fetch worker app: %s", err.Error())
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		workerApp.Stop()
	}()

	err = workerApp.Run()
	if err != nil {
		log.Fatalf("failed to run fetch worker app: %s", err.Error())
	}
}
