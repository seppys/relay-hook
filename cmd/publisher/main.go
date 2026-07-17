package main

import (
	"context"
	"log"
	"os/signal"
	"relay-hook/internal/broker"
	"relay-hook/internal/config"
	"relay-hook/internal/database"
	"relay-hook/internal/publisher"
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
