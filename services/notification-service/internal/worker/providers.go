package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/smtp"
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
	case "smtp":
		if channel != "email" {
			return logProvider{channel: channel}
		}
		host := os.Getenv("NOTIF_EMAIL_SMTP_HOST")
		port := os.Getenv("NOTIF_EMAIL_SMTP_PORT")
		user := os.Getenv("NOTIF_EMAIL_SMTP_USER")
		pass := os.Getenv("NOTIF_EMAIL_SMTP_PASS")
		from := os.Getenv("NOTIF_EMAIL_FROM")
		if host == "" || from == "" {
			return logProvider{channel: channel}
		}
		return smtpProvider{host: host, port: port, user: user, pass: pass, from: from}
	case "sms_http":
		if channel != "sms" {
			return logProvider{channel: channel}
		}
		url := os.Getenv("NOTIF_SMS_HTTP_URL")
		token := os.Getenv("NOTIF_SMS_HTTP_TOKEN")
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

type smtpProvider struct {
	host string
	port string
	user string
	pass string
	from string
}

func (p smtpProvider) Send(ctx context.Context, message, recipient string) error {
	addr := p.host
	if p.port != "" {
		addr = p.host + ":" + p.port
	}
	subject := "QMS Notification"
	body := []byte("To: " + recipient + "\r\nSubject: " + subject + "\r\n\r\n" + message)
	var auth smtp.Auth
	if p.user != "" {
		auth = smtp.PlainAuth("", p.user, p.pass, p.host)
	}
	return smtp.SendMail(addr, auth, p.from, []string{recipient}, body)
}
