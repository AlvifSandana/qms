package store

import (
	"context"
	"time"
)

type KPIResult struct {
	AvgWaitSeconds    float64 `json:"avg_wait_seconds"`
	AvgServiceSeconds float64 `json:"avg_service_seconds"`
	Count             int     `json:"count"`
}

type RealtimeResult struct {
	QueueLength int `json:"queue_length"`
	Serving     int `json:"serving"`
}

type TicketRow struct {
	TicketID    string
	Number      string
	Status      string
	CreatedAt   time.Time
	CalledAt    *time.Time
	ServedAt    *time.Time
	CompletedAt *time.Time
}

type Store interface {
	GetKPIs(ctx context.Context, tenantID, branchID, serviceID string, from, to time.Time) (KPIResult, error)
	GetRealtime(ctx context.Context, tenantID, branchID, serviceID string) (RealtimeResult, error)
	ListTickets(ctx context.Context, tenantID, branchID, serviceID string, from, to time.Time) ([]TicketRow, error)
	CreateScheduledReport(ctx context.Context, tenantID, branchID, serviceID, cron, channel, recipient string) error
	ListScheduledReports(ctx context.Context, tenantID string) ([]ScheduledReport, error)
	ListAnomalies(ctx context.Context, tenantID string) ([]Anomaly, error)
	InsertAnomaly(ctx context.Context, anomaly Anomaly) error
	ListServices(ctx context.Context) ([]ServiceRef, error)
}

type ScheduledReport struct {
	ReportID  string `json:"report_id"`
	TenantID  string `json:"tenant_id"`
	BranchID  string `json:"branch_id"`
	ServiceID string `json:"service_id"`
	Cron      string `json:"cron"`
	Channel   string `json:"channel"`
	Recipient string `json:"recipient"`
	Active    bool   `json:"active"`
}

type Anomaly struct {
	AnomalyID string  `json:"anomaly_id"`
	TenantID  string  `json:"tenant_id"`
	BranchID  string  `json:"branch_id"`
	ServiceID string  `json:"service_id"`
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	CreatedAt time.Time `json:"created_at"`
}

type ServiceRef struct {
	TenantID  string
	BranchID  string
	ServiceID string
}
