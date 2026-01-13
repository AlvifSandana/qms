package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Provider interface {
	Send(ctx context.Context, message, recipient string) error
}

func newProvider(kind string, channel string) Provider {
	switch kind {
	case "", "stub", "log":
		return logProvider{channel: channel}
	case "noop":
		return noopProvider{}
	case "fail":
		return failProvider{}
	case "webhook":
		url := os.Getenv("NOTIF_" + strings.ToUpper(channel) + "_WEBHOOK_URL")
		token := os.Getenv("NOTIF_" + strings.ToUpper(channel) + "_WEBHOOK_TOKEN")
		if url == "" {
			return logProvider{channel: channel}
		}
		return webhookProvider{channel: channel, url: url, token: token}
	default:
		if strings.HasPrefix(kind, "http://") || strings.HasPrefix(kind, "https://") {
			return webhookProvider{channel: channel, url: kind}
		}
		return logProvider{channel: channel}
	}
}

type logProvider struct {
	channel string
}

func (p logProvider) Send(ctx context.Context, message, recipient string) error {
	log.Printf("send %s to %s: %s", p.channel, recipient, message)
	return nil
}

type noopProvider struct{}

func (noopProvider) Send(ctx context.Context, message, recipient string) error {
	return nil
}

type failProvider struct{}

func (failProvider) Send(ctx context.Context, message, recipient string) error {
	return errors.New("provider failure")
}

type webhookProvider struct {
	channel string
	url     string
	token   string
}

func (p webhookProvider) Send(ctx context.Context, message, recipient string) error {
	payload := map[string]string{
		"channel":   p.channel,
		"recipient": recipient,
		"message":   message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New("provider rejected request")
	}
	return nil
}
