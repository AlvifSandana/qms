package worker

import (
	"context"
	"errors"
	"log"
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
	default:
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
