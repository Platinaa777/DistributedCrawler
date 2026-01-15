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

	workerApp, err := app.NewWorkerApp(ctx, app.SchedulerWorkerType)
	if err != nil {
		log.Fatalf("failed to init scheduler worker app: %s", err.Error())
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		workerApp.Stop()
	}()

	err = workerApp.Run()
	if err != nil {
		log.Fatalf("failed to run scheduler worker app: %s", err.Error())
	}
}
