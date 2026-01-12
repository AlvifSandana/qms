package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qms/analytics-service/internal/config"
	"qms/analytics-service/internal/httpapi"
	"qms/analytics-service/internal/store/postgres"

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

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("analytics-service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	go func() {
		if cfg.AnomalyIntervalSeconds <= 0 {
			return
		}
		ticker := time.NewTicker(time.Duration(cfg.AnomalyIntervalSeconds) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			services, err := store.ListServices(context.Background())
			if err != nil {
				log.Printf("anomaly list services error: %v", err)
				continue
			}
			windowEnd := time.Now().UTC()
			windowStart := windowEnd.Add(-1 * time.Hour)
			for _, svc := range services {
				kpi, err := store.GetKPIs(context.Background(), svc.TenantID, svc.BranchID, svc.ServiceID, windowStart, windowEnd)
				if err != nil {
					continue
				}
				if kpi.AvgWaitSeconds > cfg.AnomalyThresholdSeconds {
					_ = store.InsertAnomaly(context.Background(), store.Anomaly{
						TenantID: svc.TenantID,
						BranchID: svc.BranchID,
						ServiceID: svc.ServiceID,
						Type: "wait_time",
						Value: kpi.AvgWaitSeconds,
						Threshold: cfg.AnomalyThresholdSeconds,
					})
				}
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
