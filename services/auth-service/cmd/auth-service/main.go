package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qms/auth-service/internal/config"
	"qms/auth-service/internal/httpapi"
	"qms/auth-service/internal/store/postgres"
	"qms/auth-service/internal/telemetry"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	cfg := config.Load()
	shutdownTelemetry := telemetry.Setup("auth-service")
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTelemetry(ctx)
	}()

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

	otelHandler := otelhttp.NewHandler(httpapi.LoggingMiddleware(limiter.Middleware(handler.Routes())), "auth-service")
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      otelHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("auth-service listening on %s", server.Addr)
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
