package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimitConfig struct {
	IPPerMinute int
	IPBurst     int
}

type RateLimiter struct {
	ipLimiter *tokenLimiter
}

func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		ipLimiter: newTokenLimiter(cfg.IPPerMinute, cfg.IPBurst),
	}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if ip != "" && !l.ipLimiter.allow(ip) {
			w.WriteHeader(http.StatusTooManyRequests)
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
