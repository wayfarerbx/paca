package events

import (
	"context"
	"testing"
)

func TestTopics_AreNonEmptyAndDistinct(t *testing.T) {
	topics := []string{TopicUserCreated, TopicUserDeleted, TopicAuthLogin, TopicAuthLogout}
	seen := map[string]struct{}{}
	for _, topic := range topics {
		if topic == "" {
			t.Fatal("topic must be non-empty")
		}
		if _, ok := seen[topic]; ok {
			t.Fatalf("duplicate topic %q", topic)
		}
		seen[topic] = struct{}{}
	}
}

func TestKeys_NonEmpty(t *testing.T) {
	if ChannelRealtime == "" {
		t.Fatal("ChannelRealtime must be non-empty")
	}
	if StreamAnalytics == "" {
		t.Fatal("StreamAnalytics must be non-empty")
	}
}

type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, string, any) error        { return nil }
func (nopPublisher) Append(context.Context, string, string, any) error { return nil }

func TestPublisherInterface_Implemented(_ *testing.T) {
	var _ Publisher = nopPublisher{}
}
