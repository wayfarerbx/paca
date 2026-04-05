// Package events defines domain event publishing abstractions.
package events

import "context"

// Publisher is the application-level contract for publishing domain events.
//
// Publish sends an event over Valkey Pub/Sub for immediate real-time fan-out.
// Append writes an event to a Valkey Stream for durable analytics consumption.
type Publisher interface {
	Publish(ctx context.Context, channel string, payload any) error
	Append(ctx context.Context, stream, eventType string, payload any) error
}
