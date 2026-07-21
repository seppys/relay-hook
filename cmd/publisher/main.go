package main

import (
	"context"
	"log"
	"log/slog"
	"os/signal"
	"relay-hook/internal/broker"
	"relay-hook/internal/config"
	"relay-hook/internal/database"
	"relay-hook/internal/publisher"
	"relay-hook/internal/telemetry"
	"sync"
	"syscall"
)

func main() {
	// Environment variables
	cfg, err := config.GetConfig()

	// Context
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database
	pool, err := database.NewPool(ctx, cfg.Postgres.ConnectionString)
	if err != nil {
		log.Fatal("Error connecting to database")
	}

	otelShutdown, err := telemetry.SetupOTelSDK(ctx, "relay-hook-publisher")
	if err != nil {
		slog.Error("Error setting up OTelSDK", "err", err)
	}
	defer otelShutdown(ctx)

	// Kafka
	producer, err := broker.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	if err != nil {
		log.Fatal("Error connecting to Kafka broker")
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		publisher.Run(ctx, pool, producer)
	}()
	<-ctx.Done()
	wg.Wait()
}
