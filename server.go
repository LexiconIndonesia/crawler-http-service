package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/LexiconIndonesia/crawler-http-service/handler"
	"github.com/LexiconIndonesia/crawler-http-service/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type AppHttpServer struct {
	router     *chi.Mux
	cfg        config.Config
	server     *http.Server
	db         *db.DB
	natsClient *messaging.NatsBroker
}

func NewAppHttpServer(cfg config.Config) (*AppHttpServer, error) {
	r := chi.NewRouter()

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		// AllowedOrigins: []string{"https://bo.lexicon.id", "http://localhost:3000"},
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			// Allow all lexicon.id subdomains (*.lexicon.id)
			if origin == "https://lexicon.id" || origin == "http://lexicon.id" {
				return true
			}
			// Check for subdomains of lexicon.id
			if len(origin) > 11 { // "lexicon.id" is 10 chars, so we need at least subdomain + dot
				if origin[len(origin)-11:] == ".lexicon.id" {
					return true
				}
			}
			// Allow localhost for development
			if len(origin) >= 16 && origin[:16] == "http://localhost" {
				return true
			}
			if len(origin) >= 17 && origin[:17] == "https://localhost" {
				return true
			}
			if len(origin) >= 14 && origin[:14] == "http://127.0.0" {
				return true
			}
			if len(origin) >= 15 && origin[:15] == "https://127.0.0" {
				return true
			}
			return false
		},
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
func (s *AppHttpServer) SetNatsClient(client *messaging.NatsBroker) {
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

	// API Documentation with Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The URL pointing to API definition
	))

	// Public health endpoint (no authentication required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"crawler-http-service"}`))
	})

	r.Route("/v1", func(r chi.Router) {
		r.Use(middlewares.AccessTime())
		r.Use(middlewares.ApiKey(s.cfg.Security.BackendApiKey, s.cfg.Security.ServerSalt))
		r.Use(middlewares.RequestSignature(s.cfg.Security.ServerSalt))

		// Handlers
		crawlerHandler := handler.NewCrawlerHandler(s.db, s.natsClient, s.cfg)
		scraperHandler := handler.NewScraperHandler(s.db, s.natsClient, s.cfg)
		dataSourceHandler := handler.NewDataSourceHandler(s.db)
		workManagerHandler := handler.NewWorkManagerHandler(s.db, s.cfg)

		r.Mount("/crawlers", crawlerHandler.Router())
		r.Mount("/scrapers", scraperHandler.Router())
		r.Mount("/datasources", dataSourceHandler.Router())
		r.Mount("/works", workManagerHandler.Router())

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
