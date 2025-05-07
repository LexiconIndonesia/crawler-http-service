package module

import (
	"net/http"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/common/logger"
	"github.com/LexiconIndonesia/crawler-http-service/common/messaging"
	"github.com/go-chi/chi/v5"
)

// Module represents a module with routes
type Module struct {
	db         *db.DB
	natsClient *messaging.NatsClient
	services   map[string]interface{} // Service registry for dependency injection
	logService *logger.LogService
}

// NewModule creates a new module with dependency injection
func NewModule(db *db.DB, natsClient *messaging.NatsClient) *Module {
	// Create the module
	m := &Module{
		db:         db,
		natsClient: natsClient,
		services:   make(map[string]interface{}),
	}

	// Initialize components
	m.logService = logger.NewLogService(db)

	return m
}

// RegisterService registers a service for dependency injection
func (m *Module) RegisterService(name string, service interface{}) {
	m.services[name] = service
}

// GetService returns a registered service by name
func (m *Module) GetService(name string) interface{} {
	return m.services[name]
}

// Router returns the router for this module
func (m *Module) Router() http.Handler {
	// Create a new router
	r := chi.NewRouter()

	// Set up API routes
	apiService := NewApiService(m.db, m.natsClient)
	apiService.services = m.services // Share services
	r.Mount("/api", apiService.Routes())

	return r
}
