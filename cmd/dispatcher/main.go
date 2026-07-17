package main

import (
	"context"
	"log"
	"os/signal"
	"relay-hook/internal/broker"
	"relay-hook/internal/config"
	"relay-hook/internal/database"
	"relay-hook/internal/delivery"
	"relay-hook/internal/dispatch"
	"relay-hook/internal/subscription"
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

	subscriptionRepo := subscription.NewRepository(pool)
	subscriptionSvc := subscription.NewService(subscriptionRepo)
	deliverer := delivery.NewDeliverer()

	kafkaConsumer, err := broker.NewConsumer(cfg.Kafka.Brokers, "rh-dispatcher", cfg.Kafka.Topic)
	if err != nil {
		log.Fatal("Error connecting to Kafka broker")
	}

	dispatcher := dispatch.NewDispatcher(subscriptionSvc, deliverer)
	consumer := dispatch.NewConsumer(kafkaConsumer, dispatcher)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer.Run(ctx)
	}()
	<-ctx.Done()
	wg.Wait()
}
