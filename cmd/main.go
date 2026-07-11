package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"relay-hook/internal/broker"
	config2 "relay-hook/internal/config"
	"relay-hook/internal/database"
	"relay-hook/internal/event"
	"relay-hook/internal/publisher"
	"syscall"
	"time"
)

func main() {

	// Environment variables
	cfg, err := config2.GetConfig()

	// Context
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database
	pool, err := database.NewPool(ctx, cfg.Postgres.ConnectionString)
	if err != nil {
		log.Fatal("Error connecting to database")
	}

	// Repository
	repository := event.NewRepository(pool)

	// Kafka
	producer, err := broker.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	if err != nil {
		log.Fatal("Error connecting to Kafka broker")
	}

	go func() {
		publisher.Run(ctx, pool, producer)
	}()

	// Service
	service := event.NewService(repository)

	// HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", event.GetAllHandler(service))
	mux.HandleFunc("POST /events", event.CollectHandler(service))

	// HTTP server
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	defer srv.Close()
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()
	select {
	case err := <-serverErr:
		fmt.Printf("http server: %v", err)
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}
}
