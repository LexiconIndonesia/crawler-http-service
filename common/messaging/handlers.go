package messaging

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// MessageHandler is a function that handles a NATS message
type MessageHandler func(msg *nats.Msg) error

// JetStreamMessageHandler is a function that handles a JetStream message
type JetStreamMessageHandler func(msg jetstream.Msg) error

// SubscribeToAll subscribes to all NATS messages using the ">" wildcard
func SubscribeToAll(client *NatsClient, handler MessageHandler) (*nats.Subscription, error) {
	if client == nil || client.conn == nil {
		return nil, nil
	}

	// Create a wrapper that handles errors
	wrapperHandler := func(msg *nats.Msg) {
		if err := handler(msg); err != nil {
			log.Error().
				Err(err).
				Str("subject", msg.Subject).
				Str("data", string(msg.Data)).
				Msg("Error handling wildcard message")
		}
	}

	// Subscribe to all subjects
	sub, err := client.conn.Subscribe(">", wrapperHandler)
	if err != nil {
		return nil, err
	}

	log.Info().Msg("Subscribed to all NATS messages")
	return sub, nil
}

// SubscribeToSubject subscribes to a specific NATS subject
func SubscribeToSubject(client *NatsClient, subject string, handler MessageHandler) (*nats.Subscription, error) {
	if client == nil || client.conn == nil {
		return nil, nil
	}

	// Create a wrapper that handles errors
	wrapperHandler := func(msg *nats.Msg) {
		if err := handler(msg); err != nil {
			log.Error().
				Err(err).
				Str("subject", msg.Subject).
				Str("data", string(msg.Data)).
				Msg("Error handling message")
		}
	}

	// Subscribe to the specific subject
	sub, err := client.conn.Subscribe(subject, wrapperHandler)
	if err != nil {
		return nil, err
	}

	log.Info().Str("subject", subject).Msg("Subscribed to NATS subject")
	return sub, nil
}

// SubscribeToQueueGroup subscribes to a specific subject with a queue group
func SubscribeToQueueGroup(client *NatsClient, subject, queue string, handler MessageHandler) (*nats.Subscription, error) {
	if client == nil || client.conn == nil {
		return nil, nil
	}

	// Create a wrapper that handles errors
	wrapperHandler := func(msg *nats.Msg) {
		if err := handler(msg); err != nil {
			log.Error().
				Err(err).
				Str("subject", msg.Subject).
				Str("queue", queue).
				Str("data", string(msg.Data)).
				Msg("Error handling queue message")
		}
	}

	// Subscribe to the specific subject with queue group
	sub, err := client.conn.QueueSubscribe(subject, queue, wrapperHandler)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("subject", subject).
		Str("queue", queue).
		Msg("Subscribed to NATS subject with queue group")
	return sub, nil
}

// SubscribeToJetStream subscribes to a specific JetStream subject
func SubscribeToJetStream(client *NatsClient, streamName, subject string, handler JetStreamMessageHandler) (jetstream.Consumer, error) {
	if client == nil || client.js == nil {
		return nil, nil
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure the stream exists with the given subject
	stream, err := EnsureStream(ctx, client, streamName, []string{subject})
	if err != nil {
		return nil, err
	}

	// Create a durable consumer for the subject
	consumerName := "consumer_" + subject
	consumerConfig := jetstream.ConsumerConfig{
		Name:          consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, consumerConfig)
	if err != nil {
		return nil, err
	}

	// Create a message handler that wraps our provided handler
	msgHandler := func(msg jetstream.Msg) {
		if err := handler(msg); err != nil {
			log.Error().
				Err(err).
				Str("subject", msg.Subject()).
				Str("data", string(msg.Data())).
				Msg("Error handling JetStream message")
		} else {
			// If no error, ack the message
			if err := msg.Ack(); err != nil {
				log.Error().
					Err(err).
					Str("subject", msg.Subject()).
					Msg("Failed to acknowledge message")
			}
		}
	}

	// Start consuming messages
	_, err = consumer.Consume(msgHandler)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("stream", streamName).
		Str("subject", subject).
		Str("consumer", consumerName).
		Msg("Subscribed to JetStream subject")
	return consumer, nil
}

// SubscribeToAllJetStream subscribes to all JetStream messages in a stream
func SubscribeToAllJetStream(client *NatsClient, streamName string, handler JetStreamMessageHandler) (jetstream.Consumer, error) {
	// Same as SubscribeToJetStream but with ">" wildcard
	return SubscribeToJetStream(client, streamName, ">", handler)
}

// EnsureStream ensures a stream exists with the specified subjects
func EnsureStream(ctx context.Context, client *NatsClient, name string, subjects []string) (jetstream.Stream, error) {
	// Try to get the stream first
	stream, err := client.GetStream(ctx, name)
	if err == nil {
		return stream, nil
	}

	// If we couldn't get the stream, try to create it
	streamConfig := jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
		Storage:  jetstream.MemoryStorage,
	}

	return client.CreateStream(ctx, streamConfig)
}
