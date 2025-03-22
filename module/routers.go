package module

import (
	"net/http"

	"github.com/adryanev/go-http-service-template/common/db"
	"github.com/adryanev/go-http-service-template/common/messaging"
	"github.com/adryanev/go-http-service-template/common/utils"

	"github.com/go-chi/chi/v5"
)

// Module contains all dependencies needed for the module
type Module struct {
	DB         *db.DB
	NatsClient *messaging.NatsClient
}

// NewModule creates a new module with dependencies
func NewModule(db *db.DB, natsClient *messaging.NatsClient) *Module {
	return &Module{
		DB:         db,
		NatsClient: natsClient,
	}
}

// Router returns a router for this module
func (m *Module) Router() http.Handler {
	r := chi.NewMux()

	// Basic routes
	r.Get("/", m.testRoute)

	// Messaging routes
	r.Route("/messaging", func(r chi.Router) {
		r.Post("/publish", m.publishMessage)
		r.Get("/subscribe/{subject}", m.subscribeWebSocket)
	})

	// User routes
	r.Route("/users", func(r chi.Router) {
		r.Post("/", m.userHandler)
	})

	return r
}

// Router is a legacy function (will be deprecated)
func Router() *chi.Mux {
	r := chi.NewMux()
	r.Get("/", testRoute)
	return r
}

// @Summary Health check
// @Description Simple health check endpoint to verify the service is running
// @Tags system
// @Produce json
// @Success 200 {object} utils.Response{message=string} "Service is healthy"
// @Router / [get]
func (m *Module) testRoute(w http.ResponseWriter, r *http.Request) {
	utils.WriteMessage(w, 200, "Hello with dependency injection")
}

// testRoute is a legacy function (will be deprecated)
func testRoute(w http.ResponseWriter, req *http.Request) {
	utils.WriteMessage(w, 200, "Hello")
}
