package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimitConfig struct {
	IPPerMinute     int
	IPBurst         int
	TenantPerMinute int
	TenantBurst     int
}

type RateLimiter struct {
	ipLimiter     *tokenLimiter
	tenantLimiter *tokenLimiter
}

func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		ipLimiter:     newTokenLimiter(cfg.IPPerMinute, cfg.IPBurst),
		tenantLimiter: newTokenLimiter(cfg.TenantPerMinute, cfg.TenantBurst),
	}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if ip != "" && !l.ipLimiter.allow(ip) {
			writeError(w, "", http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}

		tenantID, requestID := extractTenantAndRequestID(r)
		if tenantID != "" && !l.tenantLimiter.allow(tenantID) {
			writeError(w, requestID, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}

		next.ServeHTTP(w, r)
	})
}

type tokenLimiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	bucket map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newTokenLimiter(perMinute, burst int) *tokenLimiter {
	if perMinute <= 0 {
		perMinute = 60
	}
	if burst <= 0 {
		burst = 20
	}
	return &tokenLimiter{
		rate:   float64(perMinute) / 60.0,
		burst:  float64(burst),
		bucket: make(map[string]*bucket),
	}
}

func (l *tokenLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.bucket[key]
	if !ok {
		l.bucket[key] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = minFloat(l.burst, b.tokens+elapsed*l.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func extractTenantAndRequestID(r *http.Request) (string, string) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	}
	if requestID == "" {
		requestID = strings.TrimSpace(r.URL.Query().Get("request_id"))
	}
	if tenantID != "" || requestID != "" || r.Body == nil {
		return tenantID, requestID
	}
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return tenantID, requestID
	}

	body, err := readBody(r)
	if err != nil {
		return tenantID, requestID
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return tenantID, requestID
	}
	if tenantID == "" {
		if value, ok := payload["tenant_id"].(string); ok {
			tenantID = strings.TrimSpace(value)
		}
	}
	if requestID == "" {
		if value, ok := payload["request_id"].(string); ok {
			requestID = strings.TrimSpace(value)
		}
	}
	return tenantID, requestID
}

func readBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}
