package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/LexiconIndonesia/crawler-http-service/common/crawler"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/logger"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/common/storage"

	"github.com/rs/zerolog/log"

	"github.com/joho/godotenv"

	_ "github.com/LexiconIndonesia/crawler-http-service/docs"
	_ "github.com/samber/lo"
	_ "github.com/samber/mo"
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

	cfg := config.DefaultConfig()
	cfg.LoadFromEnv()

	// Create a base context with cancel for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// INITIATE DATABASES
	dbConn, err := db.SetupDatabase(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup database")
	}
	defer dbConn.Close()

	// Initialize zerolog database hooks
	logger.InitializeLogging(dbConn)
	log.Info().Msg("Zerolog database hooks initialized")

	// INITIATE NATS CLIENT
	natsClient, err := messaging.SetupNatsBroker(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup NATS client")
	}
	defer natsClient.Close()

	// Setup global subscriptions
	if err := messaging.SetupGlobalSubscriptions(natsClient); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup global subscriptions")
	}

	// gcs
	gcsStorage, err := storage.NewGCSStorage(ctx, storage.GCSConfig{
		ProjectID:       cfg.GCS.ProjectID,
		CredentialsFile: cfg.GCS.CredentialsFile,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup GCS storage")
	}
	storage.SetStorageClient(gcsStorage)

	// Register all crawlers to listen to NATS messages
	if err := crawler.RegisterCrawlers(ctx, natsClient, dbConn); err != nil {
		log.Fatal().Err(err).Msg("Failed to register crawlers")
	}
	log.Info().Msg("Crawlers registered successfully")

	if err := crawler.RegisterScrapers(ctx, natsClient, dbConn); err != nil {
		log.Fatal().Err(err).Msg("Failed to register scrapers")
	}
	log.Info().Msg("Scrapers registered successfully")

	// INITIATE SERVER
	server, err := NewAppHttpServer(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create the server")
	}

	// Inject dependencies
	server.SetDB(dbConn)
	server.SetNatsClient(natsClient)

	// Register services to the module
	// mod := module.NewModule(dbConn, natsClient)
	// mod.RegisterService("crawler", crawlerService)
	// mod.RegisterService("extractor", extractorWorker)

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
