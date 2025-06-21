package messaging

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// JetStreamMessageHandler is a function that handles a JetStream message
type JetStreamMessageHandler func(msg jetstream.Msg) error

// GetJetStreamConsumer returns a JetStream consumer
func GetJetStreamConsumer(client *NatsBroker, streamName, subject string) (jetstream.Consumer, error) {
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
	consumerName := "consumer_" + strings.ReplaceAll(subject, ".", "-")
	consumerConfig := jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, consumerConfig)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("stream", streamName).
		Str("subject", subject).
		Str("consumer", consumerName).
		Msg("Got JetStream pull consumer")

	return consumer, nil
}

// EnsureStream ensures a stream exists with the specified subjects
func EnsureStream(ctx context.Context, client *NatsBroker, name string, subjects []string) (jetstream.Stream, error) {
	// Try to get the stream first
	stream, err := client.GetStream(ctx, name)
	if err != nil {
		if !errors.Is(err, jetstream.ErrStreamNotFound) && !strings.Contains(err.Error(), "stream not found") {
			log.Error().Err(err).Str("stream_name", name).Msg("Failed to get stream for unknown reasons")
			return nil, err
		}
		// If we couldn't get the stream, try to create it
		streamConfig := jetstream.StreamConfig{
			Name:     name,
			Subjects: subjects,
		}

		return client.CreateStream(ctx, streamConfig)
	}

	// Stream exists, let's update subjects if necessary
	info, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	config := info.Config
	subjectSet := make(map[string]struct{}, len(config.Subjects))
	for _, s := range config.Subjects {
		subjectSet[s] = struct{}{}
	}

	hasNewSubjects := false
	for _, s := range subjects {
		if _, ok := subjectSet[s]; !ok {
			hasNewSubjects = true
			config.Subjects = append(config.Subjects, s)
		}
	}

	if !hasNewSubjects {
		log.Debug().Str("stream_name", name).Msg("No new subjects to add to stream")
		return stream, nil
	}

	// Update the stream with the new set of subjects
	log.Info().Strs("subjects", config.Subjects).Str("stream_name", name).Msg("Updating stream with new subjects")
	return client.CreateStream(ctx, config)
}
