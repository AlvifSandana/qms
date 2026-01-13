package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qms/notification-service/internal/config"
	"qms/notification-service/internal/store/postgres"
	"qms/notification-service/internal/worker"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	w := worker.New(store, worker.Config{
		BatchSize:   cfg.BatchSize,
		MaxAttempts: cfg.MaxAttempts,
		SMSProvider: cfg.SMSProvider,
		EmailProvider: cfg.EmailProvider,
		WAProvider: cfg.WAProvider,
		PushProvider: cfg.PushProvider,
		ReminderThreshold: cfg.ReminderThreshold,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go worker.Start(ctx, cfg.PollInterval, w)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	<-shutdownCtx.Done()
}
