package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// NatsClient represents a NATS client
type NatsClient struct {
	conn        *nats.Conn
	js          jetstream.JetStream
	config      config.Config
	subscribers map[string]*nats.Subscription
	mu          sync.Mutex
}

// NewNatsClient creates a new NATS client
func NewNatsClient(config config.Config) (*NatsClient, error) {

	client := &NatsClient{
		config:      config,
		subscribers: make(map[string]*nats.Subscription),
	}

	// Connect to NATS
	if err := client.connect(); err != nil {
		return nil, err
	}

	return client, nil
}

// connect connects to the NATS server
func (c *NatsClient) connect() error {
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
	c.conn, err = nats.Connect(c.config.Nats.URL, opts...)
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
func (c *NatsClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Drain the connection (gracefully unsubscribe)
	if c.conn != nil && c.conn.IsConnected() {
		return c.conn.Drain()
	}
	return nil
}

// Publish publishes a message to a subject
func (c *NatsClient) Publish(subject string, data []byte) error {
	if c.conn == nil || !c.conn.IsConnected() {
		return fmt.Errorf("not connected to NATS")
	}

	return c.conn.Publish(subject, data)
}

// PublishAsync publishes a message to a subject asynchronously
func (c *NatsClient) PublishAsync(subject string, data []byte) (jetstream.PubAckFuture, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	// Publish to JetStream with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ack, err := c.js.PublishAsync(subject, data)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message to %s: %w", subject, err)
	}

	// Wait for ack in a goroutine
	go func() {
		select {
		case pubAck := <-ack.Ok():
			// Message was received by server
			if pubAck != nil {
				log.Debug().Str("subject", subject).
					Str("stream", pubAck.Stream).
					Uint64("seq", pubAck.Sequence).
					Msg("Message acknowledged")
			}
		case err := <-ack.Err():
			// There was an error with the message
			if err != nil {
				log.Error().Str("error", err.Error()).
					Str("subject", subject).
					Msg("Error publishing message")
			}
		case <-ctx.Done():
			// Timeout waiting for ack
			log.Warn().Str("subject", subject).
				Msg("Timeout waiting for message acknowledgement")
		}
	}()

	return ack, nil
}

// Request sends a request and waits for a response
func (c *NatsClient) Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	if c.conn == nil || !c.conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS")
	}

	return c.conn.Request(subject, data, timeout)
}

// Subscribe subscribes to a subject
func (c *NatsClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || !c.conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS")
	}

	sub, err := c.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	c.subscribers[subject] = sub
	log.Info().Str("subject", subject).Msg("Subscribed to NATS subject")
	return sub, nil
}

// QueueSubscribe subscribes to a subject with a queue group
func (c *NatsClient) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || !c.conn.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS")
	}

	sub, err := c.conn.QueueSubscribe(subject, queue, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe to %s: %w", subject, err)
	}

	c.subscribers[subject+":"+queue] = sub
	log.Info().Str("subject", subject).Str("queue", queue).Msg("Subscribed to NATS queue")
	return sub, nil
}

// CreateStream creates a JetStream stream
func (c *NatsClient) CreateStream(ctx context.Context, config jetstream.StreamConfig) (jetstream.Stream, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	stream, err := c.js.CreateOrUpdateStream(ctx, config)
	if err != nil {
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
func (c *NatsClient) GetStream(ctx context.Context, streamName string) (jetstream.Stream, error) {
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
func (c *NatsClient) CreateConsumer(ctx context.Context, streamName string, config jetstream.ConsumerConfig) (jetstream.Consumer, error) {
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
func (c *NatsClient) Consume(ctx context.Context, consumer jetstream.Consumer, handler jetstream.MessageHandler) (jetstream.ConsumeContext, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumeCtx, err := consumer.Consume(handler)
	if err != nil {
		return nil, fmt.Errorf("failed to consume from consumer: %w", err)
	}

	return consumeCtx, nil
}

// GetJetStream returns the JetStream context
func (c *NatsClient) GetJetStream() jetstream.JetStream {
	return c.js
}

// GetConn returns the NATS connection
func (c *NatsClient) GetConn() *nats.Conn {
	return c.conn
}

// setupNatsClient initializes the NATS client
func SetupNatsClient(cfg config.Config) (*NatsClient, error) {

	client, err := NewNatsClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating NATS client: %w", err)
	}

	return client, nil
}

// setupGlobalSubscriptions sets up handlers for all NATS messages
func SetupGlobalSubscriptions(natsClient *NatsClient) error {
	// Create a simple message handler function for all NATS messages
	globalHandler := func(msg *nats.Msg) error {
		log.Debug().
			Str("subject", msg.Subject).
			Str("data", string(msg.Data)).
			Msg("Received global NATS message")
		return nil
	}

	// Create a JetStream handler for persistent messages
	jsHandler := func(msg jetstream.Msg) error {
		log.Debug().
			Str("subject", msg.Subject()).
			Str("data", string(msg.Data())).
			Msg("Received global JetStream message")
		return nil
	}

	// Subscribe to all messages using the ">" wildcard
	_, err := SubscribeToAll(natsClient, globalHandler)
	if err != nil {
		return fmt.Errorf("failed to create global subscription: %w", err)
	}

	// Subscribe to specific subjects that need special handling
	_, err = SubscribeToSubject(natsClient, "notifications.*", func(msg *nats.Msg) error {
		log.Info().
			Str("subject", msg.Subject).
			Str("data", string(msg.Data)).
			Msg("Notification received")
		return nil
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create notifications subscription")
	}

	// Subscribe to JetStream messages
	// This creates a stream and consumer for all messages (using ">")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create the ALL_MESSAGES stream if it doesn't exist
	streamConfig := jetstream.StreamConfig{
		Name:     "ALL_MESSAGES",
		Subjects: []string{">"},
		Storage:  jetstream.MemoryStorage,
	}

	// Try to create the JetStream stream
	_, err = natsClient.CreateStream(ctx, streamConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create ALL_MESSAGES stream, JetStream subscription not set up")
	} else {
		// Set up JetStream subscription
		_, err = SubscribeToAllJetStream(natsClient, "ALL_MESSAGES", jsHandler)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to create JetStream subscription")
		} else {
			log.Info().Msg("Global JetStream subscription handler set up successfully")
		}
	}

	log.Info().Msg("Global NATS subscription handlers set up successfully")
	return nil
}
