package messaging

import (
	"context"
	"fmt"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// NatsBroker implements the MessageBroker interface for NATS
type NatsBroker struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	config config.Config
}

// NewNatsBroker creates a new NATS message broker
func NewNatsBroker(cfg config.Config) (*NatsBroker, error) {

	client := &NatsBroker{
		config: cfg,
	}

	// Connect to NATS
	if err := client.connect(); err != nil {
		return nil, err
	}

	return client, nil
}

// connect connects to the NATS server
func (c *NatsBroker) connect() error {
	var err error

	// Setup connection options
	opts := []nats.Option{
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Warn().Err(err).Msg("Disconnected from NATS")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info().Str("server", nc.ConnectedUrl()).Msg("Reconnected to NATS")
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Error().Err(err).
				Str("subject", sub.Subject).
				Msg("Error handling NATS message")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Info().Msg("NATS connection closed")
		}),
	}

	// Add auth if provided
	if c.config.Nats.Username != "" && c.config.Nats.Password != "" {
		opts = append(opts, nats.UserInfo(c.config.Nats.Username, c.config.Nats.Password))
	}

	// Connect to NATS
	c.conn, err = nats.Connect(c.config.Nats.URL(), opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(c.conn)
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	c.js = js

	log.Info().Str("server", c.conn.ConnectedUrl()).Msg("Connected to NATS")
	return nil
}

// Close closes the NATS connection
func (c *NatsBroker) Close() error {
	// Drain the connection (gracefully unsubscribe)
	if c.conn != nil && c.conn.IsConnected() {
		return c.conn.Drain()
	}
	return nil
}

// PublishSync publishes a message to a subject and waits for an acknowledgement
func (c *NatsBroker) PublishSync(ctx context.Context, subject string, data []byte) error {
	if c.js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	_, err := c.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message to %s: %w", subject, err)
	}

	log.Info().Str("subject", subject).Msg("Published message to NATS and received ack")

	return nil
}

// CreateStream creates a JetStream stream
func (c *NatsBroker) CreateStream(ctx context.Context, config jetstream.StreamConfig) (jetstream.Stream, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	log.Info().
		Str("name", config.Name).
		Strs("subjects", config.Subjects).
		Msg("Attempting to create or update JetStream stream")

	stream, err := c.js.CreateOrUpdateStream(ctx, config)
	if err != nil {
		log.Error().Err(err).Str("stream", config.Name).Msg("Failed to create or update stream")
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	info, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	log.Info().
		Str("name", info.Config.Name).
		Strs("subjects", info.Config.Subjects).
		Msg("Created JetStream stream")

	return stream, nil
}

// GetStream gets a JetStream stream
func (c *NatsBroker) GetStream(ctx context.Context, streamName string) (jetstream.Stream, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	stream, err := c.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	return stream, nil
}

// CreateConsumer creates a JetStream consumer
func (c *NatsBroker) CreateConsumer(ctx context.Context, streamName string, config jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	stream, err := c.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	info, err := consumer.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}

	log.Info().
		Str("name", info.Name).
		Str("stream", streamName).
		Msg("Created JetStream consumer")

	return consumer, nil
}

// Consume consumes messages from a JetStream consumer
func (c *NatsBroker) Consume(ctx context.Context, consumer jetstream.Consumer, handler jetstream.MessageHandler) (jetstream.ConsumeContext, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumeCtx, err := consumer.Consume(handler)
	if err != nil {
		return nil, fmt.Errorf("failed to consume from consumer: %w", err)
	}

	return consumeCtx, nil
}

// setupNatsBroker initializes the NATS client
func SetupNatsBroker(cfg config.Config) (*NatsBroker, error) {

	client, err := NewNatsBroker(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating NATS client: %w", err)
	}

	return client, nil
}
