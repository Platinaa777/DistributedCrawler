package main

import (
	"context"
	"distributed-crawler/internal/app"
	"log"
)

func main() {
	ctx := context.Background()

	app, err := app.NewAPIApp(ctx)
	if err != nil {
		log.Fatalf("failed to init app: %s", err.Error())
	}

	err = app.Run()
	if err != nil {
		log.Fatalf("failed to run app: %s", err.Error())
	}
}
