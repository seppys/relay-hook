package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"relay-hook/internal/auth"
	"relay-hook/internal/config"
	"relay-hook/internal/database"
	"relay-hook/internal/event"
	"relay-hook/internal/subscription"
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

	// Repository
	authRepo := auth.NewRepository(pool)
	eventRepo := event.NewRepository(pool)
	subscriptionRepo := subscription.NewRepository(pool)

	// Services
	authSvc := auth.NewService(authRepo, cfg.JWT.Secret)
	eventSvc := event.NewService(eventRepo)
	subscriptionSvc := subscription.NewService(subscriptionRepo)

	// Middlewares
	requireJWT := auth.RequireJWT(authSvc)

	// HTTP handlers
	mux := http.NewServeMux()

	// User handlers
	mux.HandleFunc("POST /users/register", auth.RegisterHandler(authSvc))
	mux.HandleFunc("POST /users/login", auth.LoginHandler(authSvc))

	// Key handlers
	mux.HandleFunc("POST /keys", requireJWT(auth.GenerateKeyHandler(authSvc)))

	// Events handlers
	mux.HandleFunc("GET /events", event.GetAllHandler(eventSvc))
	mux.HandleFunc("POST /events", event.CollectHandler(eventSvc))

	// Subscriber handlers
	mux.HandleFunc("PUT /subscribers/{id}", subscription.UpdateSubscriberHandler(subscriptionSvc))
	mux.HandleFunc("DELETE /subscribers/{id}", subscription.DeleteSubscriberHandler(subscriptionSvc))

	// Subscription handlers
	mux.HandleFunc("GET /subscriptions", subscription.GetAllHandler(subscriptionSvc))
	mux.HandleFunc("POST /subscriptions", subscription.RegisterHandler(subscriptionSvc))
	mux.HandleFunc("DELETE /subscriptions/{id}", subscription.DeleteHandler(subscriptionSvc))

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
