package httpapi

import (
	"log"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		duration := time.Since(start)
		tenantID := r.Header.Get("X-Tenant-ID")
		requestID := r.Header.Get("X-Request-ID")
		log.Printf("request method=%s path=%s status=%d duration_ms=%d tenant=%s request_id=%s", r.Method, r.URL.Path, writer.status, duration.Milliseconds(), tenantID, requestID)
	})
}
