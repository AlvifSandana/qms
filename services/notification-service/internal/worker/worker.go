package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"qms/notification-service/internal/store"

	"github.com/google/uuid"
)

type Worker struct {
	store             store.Store
	batchSize         int
	maxAttempts       int
	reminderThreshold int
	smsProvider       Provider
	emailProvider     Provider
	waProvider        Provider
	pushProvider      Provider
	prefs             notificationPrefs
	prefsPath         string
}

type payloadData map[string]interface{}

type Config struct {
	BatchSize         int
	MaxAttempts       int
	SMSProvider       string
	EmailProvider     string
	WAProvider        string
	PushProvider      string
	ReminderThreshold int
	PrefsPath         string
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
	threshold := cfg.ReminderThreshold
	if threshold <= 0 {
		threshold = 3
	}
	return &Worker{
		store:             store,
		batchSize:         batch,
		maxAttempts:       maxAttempts,
		reminderThreshold: threshold,
		smsProvider:       newProvider(cfg.SMSProvider, "sms"),
		emailProvider:     newProvider(cfg.EmailProvider, "email"),
		waProvider:        newProvider(cfg.WAProvider, "whatsapp"),
		pushProvider:      newProvider(cfg.PushProvider, "push"),
		prefs:             loadNotificationPrefs(cfg.PrefsPath),
		prefsPath:         cfg.PrefsPath,
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

	if err := w.processRetries(ctx); err != nil {
		return err
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

	if w.prefsPath != "" {
		w.prefs = loadNotificationPrefs(w.prefsPath)
	}

	templateID := templateForEvent(event.Type)
	if templateID == "" {
		return nil
	}

	if err := w.sendNotifications(ctx, event.TenantID, templateID, payload); err != nil {
		return err
	}

	if event.Type == "ticket.created" {
		if err := w.maybeSendReminder(ctx, event.TenantID, payload); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) sendNotifications(ctx context.Context, tenantID, templateID string, payload payloadData) error {
	channels := pickChannels(payload)
	if len(channels) == 0 {
		return nil
	}

	for _, channel := range channels {
		recipient := channel.recipient
		if recipient == "" {
			continue
		}
		if !w.prefs.Allow(channel.name, recipient) {
			continue
		}

		lang := "id"
		body, err := w.store.GetTemplate(ctx, tenantID, templateID, lang, channel.name)
		if err != nil {
			return err
		}
		if body == "" {
			body = defaultTemplate(templateID, lang)
		}
		message := renderTemplate(body, payload)

		notification := store.Notification{
			NotificationID: uuid.NewString(),
			TenantID:       tenantID,
			Channel:        channel.name,
			Recipient:      recipient,
			Status:         "pending",
			Attempts:       0,
			Message:        message,
		}
		if err := w.store.InsertNotification(ctx, notification); err != nil {
			return err
		}

		if err := w.deliverNotification(ctx, notification); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) maybeSendReminder(ctx context.Context, tenantID string, payload payloadData) error {
	ticketID := str(payload, "ticket_id")
	branchID := str(payload, "branch_id")
	serviceID := str(payload, "service_id")
	if ticketID == "" || branchID == "" || serviceID == "" {
		return nil
	}
	position, err := w.store.GetQueuePosition(ctx, tenantID, branchID, serviceID, ticketID)
	if err != nil {
		return err
	}
	if position <= 0 {
		return nil
	}
	ahead := position - 1
	if ahead > w.reminderThreshold {
		return nil
	}
	payload["queue_position"] = fmt.Sprint(ahead)
	return w.sendNotifications(ctx, tenantID, "ticket_reminder", payload)
}

func (w *Worker) processRetries(ctx context.Context) error {
	notifications, err := w.store.ListDueNotifications(ctx, w.batchSize)
	if err != nil {
		return err
	}
	for _, notification := range notifications {
		if err := w.deliverNotification(ctx, notification); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) deliverNotification(ctx context.Context, notification store.Notification) error {
	if notification.Message == "" {
		if _, err := w.store.MarkNotificationFailed(ctx, notification.NotificationID, "missing message"); err != nil {
			return err
		}
		if err := w.store.InsertDLQ(ctx, notification.NotificationID, "missing message"); err != nil {
			return err
		}
		return nil
	}
	provider := w.providerFor(notification.Channel)
	providerErr := provider.Send(ctx, notification.Message, notification.Recipient)
	if providerErr != nil {
		nextAttempts := notification.Attempts + 1
		if nextAttempts >= w.maxAttempts {
			if _, err := w.store.MarkNotificationFailed(ctx, notification.NotificationID, providerErr.Error()); err != nil {
				return err
			}
			if err := w.store.InsertDLQ(ctx, notification.NotificationID, "max attempts reached"); err != nil {
				return err
			}
			return nil
		}
		nextAttemptAt := time.Now().Add(retryDelay(nextAttempts))
		if _, err := w.store.MarkNotificationRetry(ctx, notification.NotificationID, providerErr.Error(), nextAttemptAt); err != nil {
			return err
		}
		return nil
	}
	if err := w.store.MarkNotificationSent(ctx, notification.NotificationID); err != nil {
		return err
	}
	return nil
}

func retryDelay(attempt int) time.Duration {
	base := 5 * time.Second
	if attempt <= 1 {
		return base
	}
	delay := base * time.Duration(1<<uint(attempt-1))
	max := 5 * time.Minute
	if delay > max {
		return max
	}
	return delay
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
		case "ticket_reminder":
			return "Ticket {ticket_number}: {queue_position} ahead."
		}
	}
	switch templateID {
	case "ticket_created":
		return "Tiket {ticket_number} dibuat."
	case "ticket_called":
		return "Tiket {ticket_number} dipanggil."
	case "ticket_recalled":
		return "Tiket {ticket_number} dipanggil ulang."
	case "ticket_reminder":
		return "Tiket {ticket_number}: {queue_position} nomor lagi."
	}
	return ""
}

func renderTemplate(template string, payload payloadData) string {
	result := template
	result = strings.ReplaceAll(result, "{ticket_number}", str(payload, "ticket_number"))
	result = strings.ReplaceAll(result, "{branch_id}", str(payload, "branch_id"))
	result = strings.ReplaceAll(result, "{service_id}", str(payload, "service_id"))
	result = strings.ReplaceAll(result, "{counter_id}", str(payload, "counter_id"))
	result = strings.ReplaceAll(result, "{queue_position}", optionalStr(payload, "queue_position"))
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

func optionalStr(payload payloadData, key string) string {
	if value, ok := payload[key]; ok {
		if text, ok := value.(string); ok {
			return text
		}
		return fmt.Sprint(value)
	}
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

func (w *Worker) providerFor(channel string) Provider {
	switch channel {
	case "sms":
		return w.smsProvider
	case "email":
		return w.emailProvider
	case "whatsapp":
		return w.waProvider
	case "push":
		return w.pushProvider
	default:
		return w.smsProvider
	}
}

type notificationPrefs struct {
	blocked map[string]map[string]struct{}
}

func loadNotificationPrefs(path string) notificationPrefs {
	if path == "" {
		return notificationPrefs{}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return notificationPrefs{}
	}
	data := map[string][]string{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return notificationPrefs{}
	}
	blocked := make(map[string]map[string]struct{})
	for channel, recipients := range data {
		channel = strings.ToLower(strings.TrimSpace(channel))
		if channel == "" {
			continue
		}
		set := make(map[string]struct{})
		for _, recipient := range recipients {
			trimmed := strings.TrimSpace(recipient)
			if trimmed == "" {
				continue
			}
			set[trimmed] = struct{}{}
		}
		if len(set) > 0 {
			blocked[channel] = set
		}
	}
	return notificationPrefs{blocked: blocked}
}

func (p notificationPrefs) Allow(channel, recipient string) bool {
	if p.blocked == nil {
		return true
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" || recipient == "" {
		return true
	}
	recipients, ok := p.blocked[channel]
	if !ok {
		return true
	}
	_, blocked := recipients[recipient]
	return !blocked
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
