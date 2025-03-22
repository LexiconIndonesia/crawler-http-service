package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/adryanev/go-http-service-template/common/db"
	"github.com/adryanev/go-http-service-template/common/messaging"
	"github.com/adryanev/go-http-service-template/module"

	_ "github.com/adryanev/go-http-service-template/docs"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type AppHttpServer struct {
	router     *chi.Mux
	cfg        config
	server     *http.Server
	db         *db.DB
	natsClient *messaging.NatsClient
}

func NewAppHttpServer(cfg config) (*AppHttpServer, error) {
	r := chi.NewRouter()

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		// AllowedOrigins: []string{"https://bo.lexicon.id", "http://localhost:3000"},
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-KEY", "X-ACCESS-TIME", "X-REQUEST-SIGNATURE", "X-API-USER", "X-REQUEST-IDENTITY"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(2 * time.Minute))

	server := &AppHttpServer{
		router: r,
		cfg:    cfg,
	}
	return server, nil
}

// SetDB sets the database dependency
func (s *AppHttpServer) SetDB(db *db.DB) {
	s.db = db
}

// SetNatsClient sets the NATS client dependency
func (s *AppHttpServer) SetNatsClient(client *messaging.NatsClient) {
	s.natsClient = client
}

func (s *AppHttpServer) setupRoute() {
	r := s.router
	// cfg := s.cfg

	// Check if dependencies are set
	if s.db == nil {
		log.Warn().Msg("DB dependency not set, using legacy global access")
	}

	if s.natsClient == nil {
		log.Warn().Msg("NATS client dependency not set")
	}

	// Create the module with dependency injection
	mod := module.NewModule(s.db, s.natsClient)

	// API Documentation with Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The URL pointing to API definition
	))

	r.Route("/v1", func(r chi.Router) {
		// r.Use(middlewares.AccessTime())
		// r.Use(middlewares.ApiKey(cfg.BackendApiKey, cfg.ServerSalt))
		// r.Use(middlewares.RequestSignature(cfg.ServerSalt))

		// Use new module structure with DI
		r.Mount("/module", mod.Router())

		// Legacy module access (will be deprecated)
		r.Mount("/legacy-module", module.Router())
	})
}

func (s *AppHttpServer) start() error {
	r := s.router
	cfg := s.cfg
	log.Info().Msg("Starting up server...")

	s.server = &http.Server{
		Addr:         cfg.Listen.Addr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// This starts the server in a goroutine from main
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// stop gracefully shuts down the server
func (s *AppHttpServer) stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
