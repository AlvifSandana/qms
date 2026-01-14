package main

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"qms/realtime-service/internal/config"
	"qms/realtime-service/internal/httpapi"
	"qms/realtime-service/internal/hub"
	"qms/realtime-service/internal/store/postgres"
	"qms/realtime-service/internal/telemetry"

	"github.com/google/uuid"
	"github.com/igm/sockjs-go/sockjs"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type eventEnvelope struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

const zeroUUID = "00000000-0000-0000-0000-000000000000"

func main() {
	cfg := config.Load()
	shutdownTelemetry := telemetry.Setup("realtime-service")
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
	h := hub.New()
	limiter := httpapi.NewRateLimiter(httpapi.RateLimitConfig{
		IPPerMinute: cfg.RateLimitPerMinute,
		IPBurst:     cfg.RateLimitBurst,
	})

	mux := http.NewServeMux()
	mux.Handle("/metrics", expvar.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	sockjsHandler := sockjs.NewHandler("/realtime", sockjs.DefaultOptions, func(session sockjs.Session) {
		req := session.Request()
		sessionID := sessionIDFromRequest(req)
		if sessionID == "" {
			_ = session.Close(4001, "missing session")
			return
		}
		authSession, err := store.GetSession(context.Background(), sessionID)
		if err != nil {
			_ = session.Close(4002, "invalid session")
			return
		}
		branches, services, err := store.GetAccess(context.Background(), authSession.UserID)
		if err != nil {
			_ = session.Close(4003, "access lookup failed")
			return
		}

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
					if !isAllowed(parsed.BranchID, parsed.ServiceID, branches, services) {
						_ = session.Close(4003, "access denied")
						return
					}
					h.UpdateSubscription(client, hub.Subscription{
						TenantID:  authSession.TenantID,
						BranchID:  parsed.BranchID,
						ServiceID: parsed.ServiceID,
					})
				}
				continue
			}
		}
	})
	mux.Handle("/realtime/", sockjsHandler)

	otelHandler := otelhttp.NewHandler(httpapi.LoggingMiddleware(limiter.Middleware(mux)), "realtime-service")
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      otelHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	offset, err := store.GetOffset(context.Background())
	if err != nil {
		log.Printf("load offset error: %v", err)
	}
	if offset.LastEventTime.IsZero() {
		offset.LastEventTime = time.Unix(0, 0).UTC()
	}
	if offset.LastEventID == "" {
		offset.LastEventID = zeroUUID
	}
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
			events, err := store.ListOutboxEvents(ctx, offset, cfg.BatchSize)
			cancel()
			if err == nil {
				for _, event := range events {
					offset.LastEventTime = event.CreatedAt
					offset.LastEventID = event.EventID
					meta := extractMeta(event.Payload)
					meta.TenantID = event.TenantID
					env := eventEnvelope{Type: event.Type, Payload: event.Payload, CreatedAt: event.CreatedAt}
					payload, _ := json.Marshal(env)
					h.Broadcast(payload, meta)
				}
				if len(events) > 0 {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					if err := store.UpdateOffset(ctx, offset); err != nil {
						log.Printf("update offset error: %v", err)
					}
					notifyOffset, err := store.GetNotificationOffset(ctx)
					if err != nil {
						log.Printf("notification offset error: %v", err)
					} else if !notifyOffset.IsZero() {
						cleanupBefore := offset.LastEventTime
						if !notifyOffset.IsZero() && notifyOffset.Before(cleanupBefore) {
							cleanupBefore = notifyOffset
						}
						if err := store.CleanupOutbox(ctx, cleanupBefore); err != nil {
							log.Printf("cleanup outbox error: %v", err)
						}
					}
					cancel()
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

func sessionIDFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if token := bearerToken(r.Header.Get("Authorization")); token != "" {
		return token
	}
	return strings.TrimSpace(r.URL.Query().Get("session_id"))
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) != 2 {
		return ""
	}
	if strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

func isAllowed(branchID, serviceID string, branches, services []string) bool {
	if len(branches) > 0 {
		if branchID == "" || !contains(branches, branchID) {
			return false
		}
	}
	if len(services) > 0 {
		if serviceID == "" || !contains(services, serviceID) {
			return false
		}
	}
	return true
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
