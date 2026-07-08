package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"relay-hook/internal/database"
	"relay-hook/internal/event"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Context
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database
	connectionString := getConnectionString()
	pool, err := database.NewPool(ctx, connectionString)
	if err != nil {
		log.Fatal("Error connecting to database")
	}

	// Repository
	repository := event.NewRepository(pool)

	// Service
	service := event.NewService(repository)

	// HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", event.GetAllHandler(service))
	mux.HandleFunc("POST /events", event.CollectHandler(service))

	// HTTP server
	srv := &http.Server{
		Addr:              "127.0.0.1:8000",
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

// todo: add validation field by field
func getConnectionString() string {
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_DATABASE")
	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")
	connectionString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbUsername, dbPassword, dbHost, dbName)
	return connectionString
}
