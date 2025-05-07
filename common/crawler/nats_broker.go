package crawler

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// Errors
var (
	ErrNotImplemented = errors.New("method not implemented")
)

// NatsBroker implements the MessageBroker interface for NATS
type NatsBroker struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

// NewNatsBroker creates a new NATS message broker
func NewNatsBroker() (*NatsBroker, error) {
	// This is just a stub - no actual implementation
	return &NatsBroker{}, nil
}

// Publish publishes a message to a topic
func (n *NatsBroker) Publish(ctx context.Context, topic string, msg []byte) error {
	// Not implemented as per requirements
	log.Error().Msg("Publish method not implemented")
	return ErrNotImplemented
}

// Subscribe subscribes to a topic with a handler
func (n *NatsBroker) Subscribe(ctx context.Context, topic string, handler func([]byte) error) error {
	// Not implemented as per requirements
	log.Error().Msg("Subscribe method not implemented")
	return ErrNotImplemented
}

// Close closes the NATS connection
func (n *NatsBroker) Close(ctx context.Context) error {
	// Not implemented as per requirements
	log.Error().Msg("Close method not implemented")
	return ErrNotImplemented
}
