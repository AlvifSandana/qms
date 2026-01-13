package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"qms/analytics-service/internal/config"
	"qms/analytics-service/internal/httpapi"
	"qms/analytics-service/internal/store"
	"qms/analytics-service/internal/store/postgres"
	"qms/analytics-service/internal/telemetry"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	cfg := config.Load()
	shutdownTelemetry := telemetry.Setup("analytics-service")
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

	repo := postgres.NewStore(pool)
	handler := httpapi.NewHandler(repo, httpapi.Options{BIAPIToken: cfg.BIAPIToken})
	limiter := httpapi.NewRateLimiter(httpapi.RateLimitConfig{
		IPPerMinute:     cfg.RateLimitPerMinute,
		IPBurst:         cfg.RateLimitBurst,
		TenantPerMinute: cfg.TenantRateLimitPerMinute,
		TenantBurst:     cfg.TenantRateLimitBurst,
	})

	otelHandler := otelhttp.NewHandler(httpapi.LoggingMiddleware(limiter.Middleware(handler.Routes())), "analytics-service")
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      otelHandler,
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
			services, err := repo.ListServices(context.Background())
			if err != nil {
				log.Printf("anomaly list services error: %v", err)
				continue
			}
			windowEnd := time.Now().UTC()
			windowStart := windowEnd.Add(-1 * time.Hour)
			for _, svc := range services {
				kpi, err := repo.GetKPIs(context.Background(), svc.TenantID, svc.BranchID, svc.ServiceID, windowStart, windowEnd)
				if err != nil {
					continue
				}
				if kpi.AvgWaitSeconds > cfg.AnomalyThresholdSeconds {
					_ = repo.InsertAnomaly(context.Background(), store.Anomaly{
						TenantID:  svc.TenantID,
						BranchID:  svc.BranchID,
						ServiceID: svc.ServiceID,
						Type:      "wait_time",
						Value:     kpi.AvgWaitSeconds,
						Threshold: cfg.AnomalyThresholdSeconds,
					})
				}
			}
		}
	}()

	go func() {
		if cfg.ReportIntervalSeconds <= 0 {
			return
		}
		ticker := time.NewTicker(time.Duration(cfg.ReportIntervalSeconds) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if cfg.ReportWebhookURL == "" {
				continue
			}
			if err := runScheduledReports(context.Background(), repo, cfg); err != nil {
				log.Printf("report worker error: %v", err)
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

func runScheduledReports(ctx context.Context, repo store.Store, cfg config.Config) error {
	reports, err := repo.ListScheduledReports(ctx, "")
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, report := range reports {
		if !report.Active {
			continue
		}
		interval, err := parseReportInterval(report.Cron)
		if err != nil {
			continue
		}
		if report.LastSentAt != nil && now.Sub(*report.LastSentAt) < interval {
			continue
		}
		from := now.Add(-24 * time.Hour)
		if report.LastSentAt != nil {
			from = *report.LastSentAt
		}
		to := now
		tickets, err := repo.ListTickets(ctx, report.TenantID, report.BranchID, report.ServiceID, from, to)
		if err != nil {
			log.Printf("report list tickets error: %v", err)
			continue
		}
		csvData, err := buildCSV(tickets)
		if err != nil {
			log.Printf("report csv error: %v", err)
			continue
		}
		if err := sendReport(cfg, report, from, to, csvData); err != nil {
			log.Printf("report send error: %v", err)
			continue
		}
		if err := repo.UpdateScheduledReportSent(ctx, report.ReportID, now); err != nil {
			log.Printf("report update error: %v", err)
		}
	}
	return nil
}

func parseReportInterval(cron string) (time.Duration, error) {
	trimmed := strings.TrimSpace(cron)
	if trimmed == "" {
		return 0, errors.New("empty schedule")
	}
	if strings.HasPrefix(trimmed, "@every ") {
		trimmed = strings.TrimPrefix(trimmed, "@every ")
	}
	return time.ParseDuration(trimmed)
}

func buildCSV(rows []store.TicketRow) ([]byte, error) {
	buf := &strings.Builder{}
	writer := csv.NewWriter(buf)
	_ = writer.Write([]string{"ticket_id", "ticket_number", "status", "created_at", "called_at", "served_at", "completed_at"})
	for _, row := range rows {
		_ = writer.Write([]string{
			row.TicketID,
			row.Number,
			row.Status,
			row.CreatedAt.Format(time.RFC3339),
			formatTime(row.CalledAt),
			formatTime(row.ServedAt),
			formatTime(row.CompletedAt),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func sendReport(cfg config.Config, report store.ScheduledReport, from, to time.Time, csvData []byte) error {
	if strings.EqualFold(report.Channel, "email") {
		return sendReportEmail(cfg, report, from, to, csvData)
	}
	return sendReportWebhook(cfg, report, from, to, csvData)
}

func sendReportWebhook(cfg config.Config, report store.ScheduledReport, from, to time.Time, csvData []byte) error {
	if cfg.ReportWebhookURL == "" {
		return errors.New("report webhook is not configured")
	}
	payload := map[string]interface{}{
		"report_id":  report.ReportID,
		"tenant_id":  report.TenantID,
		"branch_id":  report.BranchID,
		"service_id": report.ServiceID,
		"channel":    report.Channel,
		"recipient":  report.Recipient,
		"from":       from.Format(time.RFC3339),
		"to":         to.Format(time.RFC3339),
		"csv":        string(csvData),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, cfg.ReportWebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.ReportWebhookToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.ReportWebhookToken)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("report webhook status %d", resp.StatusCode)
	}
	return nil
}

func sendReportEmail(cfg config.Config, report store.ScheduledReport, from, to time.Time, csvData []byte) error {
	if cfg.ReportEmailHost == "" || cfg.ReportEmailFrom == "" {
		return errors.New("report email is not configured")
	}
	subject := cfg.ReportEmailSubject
	if subject == "" {
		subject = "QMS Scheduled Report"
	}
	body := fmt.Sprintf("Report %s\nTenant: %s\nBranch: %s\nService: %s\nFrom: %s\nTo: %s\n\n%s",
		report.ReportID,
		report.TenantID,
		report.BranchID,
		report.ServiceID,
		from.Format(time.RFC3339),
		to.Format(time.RFC3339),
		string(csvData),
	)
	message := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", report.Recipient, subject, body))
	addr := fmt.Sprintf("%s:%d", cfg.ReportEmailHost, cfg.ReportEmailPort)
	var auth smtp.Auth
	if cfg.ReportEmailUser != "" {
		auth = smtp.PlainAuth("", cfg.ReportEmailUser, cfg.ReportEmailPass, cfg.ReportEmailHost)
	}
	return smtp.SendMail(addr, auth, cfg.ReportEmailFrom, []string{report.Recipient}, message)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}
