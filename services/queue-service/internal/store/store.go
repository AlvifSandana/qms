package store

import (
	"context"
	"encoding/json"
	"time"

	"qms/queue-service/internal/models"
)

type CreateTicketInput struct {
	RequestID     string
	TenantID      string
	BranchID      string
	ServiceID     string
	AreaID        string
	Channel       string
	PriorityClass string
	Phone         string
	CreatedAt     time.Time
}

type CallNextInput struct {
	RequestID string
	TenantID  string
	BranchID  string
	ServiceID string
	CounterID string
	CalledAt  time.Time
}

type TicketActionInput struct {
	RequestID     string
	TenantID      string
	BranchID      string
	TicketID      string
	CounterID     string
	ServiceID     string
	Reason        string
	OccurredAt    time.Time
	ReturnToQueue bool
}

type TicketStore interface {
	CreateTicket(ctx context.Context, input CreateTicketInput) (models.Ticket, bool, error)
	GetTicket(ctx context.Context, tenantID, branchID, ticketID string) (models.Ticket, bool, error)
	ListQueue(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error)
	CallNext(ctx context.Context, input CallNextInput) (models.Ticket, bool, error)
	StartServing(ctx context.Context, input TicketActionInput) (models.Ticket, bool, error)
	CompleteTicket(ctx context.Context, input TicketActionInput) (models.Ticket, bool, error)
	CancelTicket(ctx context.Context, input TicketActionInput) (models.Ticket, bool, error)
	RecallTicket(ctx context.Context, input TicketActionInput) (models.Ticket, bool, error)
	HoldTicket(ctx context.Context, input TicketActionInput) (models.Ticket, bool, error)
	UnholdTicket(ctx context.Context, input TicketActionInput) (models.Ticket, bool, error)
	TransferTicket(ctx context.Context, input TicketActionInput) (models.Ticket, bool, error)
	NoShowTicket(ctx context.Context, input TicketActionInput) (models.Ticket, bool, error)
	SnapshotTickets(ctx context.Context, tenantID, branchID, serviceID string) ([]models.Ticket, error)
	GetActiveTicket(ctx context.Context, tenantID, branchID, counterID string) (models.Ticket, bool, error)
	ListOutboxEvents(ctx context.Context, tenantID string, after time.Time, limit int) ([]OutboxEvent, error)
	ListTicketEvents(ctx context.Context, tenantID, ticketID string) ([]TicketEvent, error)
	ListCounters(ctx context.Context, tenantID, branchID string) ([]models.Counter, error)
	UpdateCounterStatus(ctx context.Context, tenantID, branchID, counterID, status string) error
	ListServices(ctx context.Context, tenantID, branchID string) ([]models.Service, error)
	CheckInAppointment(ctx context.Context, requestID, tenantID, branchID, appointmentID string) (models.Ticket, error)
	GetSession(ctx context.Context, sessionID string) (Session, error)
	GetAccess(ctx context.Context, userID string) ([]string, []string, error)
}

type Session struct {
	SessionID string
	UserID    string
	TenantID  string
	Role      string
	ExpiresAt time.Time
}

type OutboxEvent struct {
	EventID   string          `json:"event_id"`
	TenantID  string          `json:"tenant_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}
