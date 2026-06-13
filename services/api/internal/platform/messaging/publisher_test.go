package messaging

import (
	"context"
	"strings"
	"testing"
)

func TestPublish_NotInitialized(t *testing.T) {
	p := &Publisher{}

	err := p.Publish(context.Background(), "paca.events", struct{}{})
	if err == nil {
		t.Fatal("expected not-initialized error")
	}
	if !strings.Contains(err.Error(), "messaging: publisher not initialized") {
		t.Fatalf("expected not-initialized error, got %v", err)
	}
}

func TestAppend_NotInitialized(t *testing.T) {
	p := &Publisher{}

	err := p.Append(context.Background(), "paca.analytics", "user.created", struct{}{})
	if err == nil {
		t.Fatal("expected not-initialized error")
	}
	if !strings.Contains(err.Error(), "messaging: publisher not initialized") {
		t.Fatalf("expected not-initialized error, got %v", err)
	}
}

func TestClose_NilSafe(_ *testing.T) {
	var p *Publisher
	p.Close()

	(&Publisher{}).Close()
}
