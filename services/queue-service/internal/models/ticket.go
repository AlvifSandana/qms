package models

import "time"

type Ticket struct {
	TicketID     string     `json:"ticket_id"`
	TicketNumber string     `json:"ticket_number"`
	TenantID     string     `json:"tenant_id,omitempty"`
	BranchID     string     `json:"branch_id,omitempty"`
	ServiceID    string     `json:"service_id,omitempty"`
	AreaID       string     `json:"area_id,omitempty"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	RequestID    string     `json:"request_id"`
	CalledAt     *time.Time `json:"called_at,omitempty"`
	CounterID    *string    `json:"counter_id,omitempty"`
	ServedAt     *time.Time `json:"served_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Phone        string     `json:"phone,omitempty"`
}

const (
	StatusWaiting   = "waiting"
	StatusCalled    = "called"
	StatusServing   = "serving"
	StatusDone      = "done"
	StatusNoShow    = "no_show"
	StatusCancelled = "cancelled"
	StatusHeld      = "held"
)
