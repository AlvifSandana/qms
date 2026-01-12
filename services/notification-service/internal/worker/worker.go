package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"qms/notification-service/internal/store"

	"github.com/google/uuid"
)

type Worker struct {
	store       store.Store
	batchSize   int
	maxAttempts int
	smsProvider string
	emailProvider string
	waProvider string
	pushProvider string
}

type payloadData map[string]interface{}

type Config struct {
	BatchSize   int
	MaxAttempts int
	SMSProvider string
	EmailProvider string
	WAProvider string
	PushProvider string
}

func New(store store.Store, cfg Config) *Worker {
	batch := cfg.BatchSize
	if batch <= 0 {
		batch = 50
	}
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return &Worker{
		store: store,
		batchSize: batch,
		maxAttempts: maxAttempts,
		smsProvider: cfg.SMSProvider,
		emailProvider: cfg.EmailProvider,
		waProvider: cfg.WAProvider,
		pushProvider: cfg.PushProvider,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	last, err := w.store.GetLastOffset(ctx)
	if err != nil {
		return err
	}

	events, err := w.store.ListOutboxEvents(ctx, last, w.batchSize)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := w.processEvent(ctx, event); err != nil {
			log.Printf("notif process error: %v", err)
		}
		last = event.CreatedAt
	}

	if !last.IsZero() {
		if err := w.store.UpdateOffset(ctx, last); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) processEvent(ctx context.Context, event store.OutboxEvent) error {
	enabled, err := w.store.IsNotificationsEnabled(ctx, event.TenantID)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}

	payload := payloadData{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	templateID := templateForEvent(event.Type)
	if templateID == "" {
		return nil
	}

	channels := pickChannels(payload)
	if len(channels) == 0 {
		return nil
	}

	for _, channel := range channels {
		recipient := channel.recipient
		if recipient == "" {
			continue
		}

		lang := "id"
		body, err := w.store.GetTemplate(ctx, event.TenantID, templateID, lang, channel.name)
		if err != nil {
			return err
		}
		if body == "" {
			body = defaultTemplate(templateID, lang)
		}
		message := renderTemplate(body, payload)

		notification := store.Notification{
			NotificationID: uuid.NewString(),
			TenantID:  event.TenantID,
			Channel:   channel.name,
			Recipient: recipient,
			Status:    "pending",
			Attempts:  1,
		}
		if err := w.store.InsertNotification(ctx, notification); err != nil {
			return err
		}

		providerErr := w.send(channel.name, message, recipient)
		if providerErr != nil {
			if err := w.store.MarkNotificationFailed(ctx, notification.NotificationID, providerErr.Error()); err != nil {
				return err
			}
			if notification.Attempts >= w.maxAttempts {
				if err := w.store.InsertDLQ(ctx, notification.NotificationID, "max attempts reached"); err != nil {
					return err
				}
			}
			continue
		}
		if err := w.store.MarkNotificationSent(ctx, notification.NotificationID); err != nil {
			return err
		}
	}
	return nil
}

func templateForEvent(eventType string) string {
	switch eventType {
	case "ticket.created":
		return "ticket_created"
	case "ticket.called":
		return "ticket_called"
	case "ticket.recalled":
		return "ticket_recalled"
	default:
		return ""
	}
}

func defaultTemplate(templateID, lang string) string {
	if lang == "en" {
		switch templateID {
		case "ticket_created":
			return "Ticket {ticket_number} created."
		case "ticket_called":
			return "Ticket {ticket_number} called."
		case "ticket_recalled":
			return "Ticket {ticket_number} recalled."
		}
	}
	switch templateID {
	case "ticket_created":
		return "Tiket {ticket_number} dibuat."
	case "ticket_called":
		return "Tiket {ticket_number} dipanggil."
	case "ticket_recalled":
		return "Tiket {ticket_number} dipanggil ulang."
	}
	return ""
}

func renderTemplate(template string, payload payloadData) string {
	result := template
	result = strings.ReplaceAll(result, "{ticket_number}", str(payload, "ticket_number"))
	result = strings.ReplaceAll(result, "{branch_id}", str(payload, "branch_id"))
	result = strings.ReplaceAll(result, "{service_id}", str(payload, "service_id"))
	result = strings.ReplaceAll(result, "{counter_id}", str(payload, "counter_id"))
	return result
}

func str(payload payloadData, key string) string {
	if value, ok := payload[key]; ok {
		if text, ok := value.(string); ok {
			return text
		}
	}
	log.Printf("notif missing variable: %s", key)
	return ""
}

func findRecipient(payload payloadData) string {
	if email, ok := payload["email"].(string); ok && email != "" {
		return email
	}
	if phone, ok := payload["phone"].(string); ok && phone != "" {
		return phone
	}
	return ""
}

type channelTarget struct {
	name      string
	recipient string
}

func pickChannels(payload payloadData) []channelTarget {
	var channels []channelTarget
	if phone, ok := payload["phone"].(string); ok && phone != "" {
		channels = append(channels, channelTarget{name: "sms", recipient: phone})
	}
	if email, ok := payload["email"].(string); ok && email != "" {
		channels = append(channels, channelTarget{name: "email", recipient: email})
	}
	if wa, ok := payload["whatsapp"].(string); ok && wa != "" {
		channels = append(channels, channelTarget{name: "whatsapp", recipient: wa})
	}
	if token, ok := payload["device_token"].(string); ok && token != "" {
		channels = append(channels, channelTarget{name: "push", recipient: token})
	}
	return channels
}

func (w *Worker) send(channel, message, recipient string) error {
	provider := ""
	switch channel {
	case "sms":
		provider = w.smsProvider
	case "email":
		provider = w.emailProvider
	case "whatsapp":
		provider = w.waProvider
	case "push":
		provider = w.pushProvider
	}
	if strings.Contains(recipient, "fail") {
		return errors.New("provider failure")
	}
	log.Printf("send %s via %s to %s: %s", channel, provider, recipient, message)
	return nil
}

func Start(ctx context.Context, interval time.Duration, w *Worker) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Run(ctx); err != nil {
				log.Printf("notif worker error: %v", err)
			}
		}
	}
}
