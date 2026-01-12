package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"qms/realtime-service/internal/config"
	"qms/realtime-service/internal/hub"
	"qms/realtime-service/internal/store/postgres"

	"github.com/google/uuid"
	"github.com/igm/sockjs-go/sockjs"
	"github.com/jackc/pgx/v5/pgxpool"
)

type eventEnvelope struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func main() {
	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	h := hub.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/realtime/info", sockjs.InfoHandler)
	mux.Handle("/realtime/", sockjs.NewHandler("/realtime", sockjs.DefaultOptions, func(session sockjs.Session) {
		client := &hub.Client{ID: uuid.NewString(), Send: make(chan []byte, 16)}
		h.Register(client)
		defer h.Unregister(client)

		go func() {
			for msg := range client.Send {
				_ = session.Send(string(msg))
			}
		}()

		for {
			msg, err := session.Recv()
			if err != nil {
				return
			}
			parsed, ok := hub.ParseSubscribe([]byte(msg))
			if ok {
				if parsed.Action == "unsubscribe" {
					h.UpdateSubscription(client, hub.Subscription{})
				} else {
					h.UpdateSubscription(client, hub.Subscription{TenantID: parsed.TenantID, BranchID: parsed.BranchID, ServiceID: parsed.ServiceID})
				}
				continue
			}
		}
	}))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	var last time.Time
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	var running int32

	go func() {
		log.Printf("realtime-service listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.PollInterval)
		defer ticker.Stop()
		for range ticker.C {
			if !atomic.CompareAndSwapInt32(&running, 0, 1) {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			events, err := store.ListOutboxEvents(ctx, last, cfg.BatchSize)
			cancel()
			if err == nil {
				for _, event := range events {
					last = event.CreatedAt
					meta := extractMeta(event.Payload)
					meta.TenantID = event.TenantID
					env := eventEnvelope{Type: event.Type, Payload: event.Payload, CreatedAt: event.CreatedAt}
					payload, _ := json.Marshal(env)
					h.Broadcast(payload, meta)
				}
			}
			atomic.StoreInt32(&running, 0)
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

func extractMeta(payload []byte) hub.Subscription {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return hub.Subscription{}
	}
	return hub.Subscription{
		TenantID:  str(data["tenant_id"]),
		BranchID:  str(data["branch_id"]),
		ServiceID: str(data["service_id"]),
	}
}

func str(value interface{}) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	return fmt.Sprint(value)
}
