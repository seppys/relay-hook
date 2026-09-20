package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"relay-hook/internal/broker"
	"relay-hook/internal/config"
	"relay-hook/internal/database"
	"relay-hook/internal/delivery"
	"relay-hook/internal/dispatch"
	"relay-hook/internal/subscription"
	"sync"
	"syscall"
	"time"
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
	deliveryRepo := delivery.NewRepository(pool)
	subscriptionSvc := subscription.NewService(subscriptionRepo)
	deliveryScv := delivery.NewService(deliveryRepo, &http.Client{
		Timeout: 10 * time.Second,
	})

	kafkaConsumer, err := broker.NewConsumer(cfg.Kafka.Brokers, "rh-dispatcher", cfg.Kafka.Topic)
	if err != nil {
		log.Fatal("Error connecting to Kafka broker")
	}

	dispatcher := dispatch.NewDispatcher(subscriptionSvc, *deliveryScv)
	consumer := dispatch.NewConsumer(kafkaConsumer, dispatcher)
	retrier := delivery.NewRetrier(*deliveryScv)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		retrier.Run(ctx)
	}()

	<-ctx.Done()
	wg.Wait()
}
