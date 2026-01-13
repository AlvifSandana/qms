package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"qms/queue-service/internal/models"
)

type TicketEvent struct {
	TicketID  string          `json:"ticket_id"`
	TicketSeq int             `json:"ticket_seq"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	PrevHash  string          `json:"prev_hash"`
	Hash      string          `json:"hash"`
}

type eventPayload struct {
	TicketID      string     `json:"ticket_id"`
	TicketNumber  string     `json:"ticket_number"`
	Status        string     `json:"status"`
	TenantID      string     `json:"tenant_id"`
	BranchID      string     `json:"branch_id"`
	ServiceID     string     `json:"service_id"`
	FromServiceID string     `json:"from_service_id"`
	ToServiceID   string     `json:"to_service_id"`
	CreatedAt     *time.Time `json:"created_at"`
	CalledAt      *time.Time `json:"called_at"`
	ServedAt      *time.Time `json:"served_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	CounterID     *string    `json:"counter_id"`
}

func ComputeTicketEventHash(prevHash, ticketID, eventType string, payload json.RawMessage, createdAt time.Time, seq int) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%d|%s", prevHash, ticketID, eventType, createdAt.UTC().Format(time.RFC3339Nano), seq, payload)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

func RehydrateTicket(events []TicketEvent) (models.Ticket, error) {
	var ticket models.Ticket
	for _, event := range events {
		if len(event.Payload) == 0 {
			continue
		}
		var payload eventPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return models.Ticket{}, err
		}
		if payload.TicketID != "" {
			ticket.TicketID = payload.TicketID
		}
		if payload.TicketNumber != "" {
			ticket.TicketNumber = payload.TicketNumber
		}
		if payload.TenantID != "" {
			ticket.TenantID = payload.TenantID
		}
		if payload.BranchID != "" {
			ticket.BranchID = payload.BranchID
		}
		if payload.ServiceID != "" {
			ticket.ServiceID = payload.ServiceID
		}
		if payload.ToServiceID != "" {
			ticket.ServiceID = payload.ToServiceID
		}
		if payload.Status != "" {
			ticket.Status = payload.Status
		}
		if payload.CreatedAt != nil {
			ticket.CreatedAt = *payload.CreatedAt
		}
		if payload.CalledAt != nil {
			ticket.CalledAt = payload.CalledAt
		}
		if payload.ServedAt != nil {
			ticket.ServedAt = payload.ServedAt
		}
		if payload.CompletedAt != nil {
			ticket.CompletedAt = payload.CompletedAt
		}
		if payload.CounterID != nil {
			ticket.CounterID = payload.CounterID
		}
	}
	return ticket, nil
}
