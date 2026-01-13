package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qms/admin-service/internal/config"
	"qms/admin-service/internal/httpapi"
	"qms/admin-service/internal/store/postgres"

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
	handler := httpapi.NewHandler(store)
	limiter := httpapi.NewRateLimiter(httpapi.RateLimitConfig{
		IPPerMinute:     cfg.RateLimitPerMinute,
		IPBurst:         cfg.RateLimitBurst,
		TenantPerMinute: cfg.TenantRateLimitPerMinute,
		TenantBurst:     cfg.TenantRateLimitBurst,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httpapi.LoggingMiddleware(limiter.Middleware(handler.Routes())),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("admin-service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
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
