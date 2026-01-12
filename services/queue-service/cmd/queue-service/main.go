package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qms/queue-service/internal/config"
	"qms/queue-service/internal/httpapi"
	"qms/queue-service/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	store := postgres.NewStore(pool, postgres.Options{
		NoShowReturnToQueue: cfg.NoShowReturnToQueue,
		PriorityStreakLimit: cfg.PriorityStreakLimit,
	})
	handler := httpapi.NewHandler(store, httpapi.Options{
		NoShowReturnToQueue: cfg.NoShowReturnToQueue,
	})
	limiter := httpapi.NewRateLimiter(httpapi.RateLimitConfig{
		IPPerMinute:     cfg.RateLimitPerMinute,
		IPBurst:         cfg.RateLimitBurst,
		TenantPerMinute: cfg.TenantRateLimitPerMinute,
		TenantBurst:     cfg.TenantRateLimitBurst,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      limiter.Middleware(handler.Routes()),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("queue-service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	go func() {
		if cfg.NoShowGrace <= 0 || cfg.NoShowInterval <= 0 {
			return
		}
		ticker := time.NewTicker(cfg.NoShowInterval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			count, err := store.AutoNoShow(ctx, cfg.NoShowGrace, cfg.NoShowBatchSize)
			cancel()
			if err != nil {
				log.Printf("auto no-show error: %v", err)
				continue
			}
			if count > 0 {
				log.Printf("auto no-show processed %d tickets", count)
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
