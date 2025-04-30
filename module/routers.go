package module

import (
	"net/http"

	"github.com/adryanev/go-http-service-template/common/utils"

	"github.com/go-chi/chi/v5"
)

// SetupBaseRoutes returns base routes for this module
func (m *Module) SetupBaseRoutes() http.Handler {
	r := chi.NewMux()

	// Basic routes
	r.Get("/", m.testRoute)

	// Messaging routes
	r.Route("/messaging", func(r chi.Router) {
		r.Post("/publish", m.publishMessage)
		r.Get("/subscribe/{subject}", m.subscribeWebSocket)
	})

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
