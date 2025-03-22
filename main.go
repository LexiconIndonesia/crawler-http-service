package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adryanev/go-http-service-template/common/db"
	"github.com/adryanev/go-http-service-template/common/messaging"
	"github.com/adryanev/go-http-service-template/repository"

	"github.com/rs/zerolog/log"

	zerolog "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/joho/godotenv"

	_ "github.com/adryanev/go-http-service-template/docs"
	_ "github.com/samber/lo"
	_ "github.com/samber/mo"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// @title          Go HTTP Service API
// @version        1.0
// @description    API documentation for Go HTTP Service Template
// @termsOfService http://swagger.io/terms/

// @contact.name  API Support
// @contact.url   http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url  http://www.apache.org/licenses/LICENSE-2.0.html

// @host     localhost:8080
// @BasePath /v1
// @schemes  http https

// @securityDefinitions.apikey ApiKeyAuth
// @in                         header
// @name                       X-API-KEY

func main() {
	// INITIATE CONFIGURATION
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("Error loading .env file, using environment variables")
	}

	cfg := defaultConfig()
	cfg.loadFromEnv()

	// Create a base context with cancel for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// INITIATE DATABASES
	dbConn, err := setupDatabase(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup database")
	}
	defer dbConn.Close()

	// INITIATE NATS CLIENT
	natsClient, err := setupNatsClient(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup NATS client")
	}
	defer natsClient.Close()

	// Setup global subscriptions
	if err := setupGlobalSubscriptions(natsClient); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup global subscriptions")
	}

	// INITIATE SERVER
	server, err := NewAppHttpServer(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create the server")
	}

	// Inject dependencies
	server.SetDB(dbConn)
	server.SetNatsClient(natsClient)

	// Setup routes
	server.setupRoute()

	// Start server in a goroutine
	go func() {
		if err := server.start(); err != nil {
			log.Error().Err(err).Msg("Server error")
			cancel()
		}
	}()

	log.Info().Str("address", cfg.Listen.Addr()).Msg("Server started successfully")
	log.Info().Str("swagger", fmt.Sprintf("http://%s/swagger/index.html", cfg.Listen.Addr())).Msg("Swagger documentation available at")

	// Wait for shutdown signal
	<-shutdown
	log.Info().Msg("Shutdown signal received")

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.stop(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
	}

	log.Info().Msg("Server gracefully stopped")
}

// setupDatabase initializes the database connection
func setupDatabase(ctx context.Context, cfg config) (*db.DB, error) {
	config, err := pgxpool.ParseConfig(cfg.PgSql.ConnStr())
	if err != nil {
		return nil, fmt.Errorf("parsing database config: %w", err)
	}

	// Setup logger
	logger := zerolog.NewLogger(log.Logger)
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelInfo,
	}

	pgsqlClient, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Test connection
	if err := pgsqlClient.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	queries := repository.New(pgsqlClient)

	// Create DB struct for dependency injection
	dbConn, err := db.New(pgsqlClient, queries)
	if err != nil {
		return nil, fmt.Errorf("creating DB handler: %w", err)
	}

	return dbConn, nil
}

// setupNatsClient initializes the NATS client
func setupNatsClient(cfg config) (*messaging.NatsClient, error) {
	natsConfig := messaging.Config{
		URL:      cfg.Nats.URL,
		Username: cfg.Nats.Username,
		Password: cfg.Nats.Password,
	}

	client, err := messaging.NewNatsClient(natsConfig)
	if err != nil {
		return nil, fmt.Errorf("creating NATS client: %w", err)
	}

	return client, nil
}

// setupGlobalSubscriptions sets up handlers for all NATS messages
func setupGlobalSubscriptions(natsClient *messaging.NatsClient) error {
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
	_, err := messaging.SubscribeToAll(natsClient, globalHandler)
	if err != nil {
		return fmt.Errorf("failed to create global subscription: %w", err)
	}

	// Subscribe to specific subjects that need special handling
	_, err = messaging.SubscribeToSubject(natsClient, "notifications.*", func(msg *nats.Msg) error {
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
		_, err = messaging.SubscribeToAllJetStream(natsClient, "ALL_MESSAGES", jsHandler)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to create JetStream subscription")
		} else {
			log.Info().Msg("Global JetStream subscription handler set up successfully")
		}
	}

	log.Info().Msg("Global NATS subscription handlers set up successfully")
	return nil
}
