package crawler

import (
	"context"
)

// MessageBroker defines the interface for messaging operations
type MessageBroker interface {
	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, msg []byte) error

	// Subscribe subscribes to a topic with a handler
	Subscribe(ctx context.Context, topic string, handler func([]byte) error) error

	// Close closes the message broker connection
	Close(ctx context.Context) error
}
