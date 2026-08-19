package http

import (
	"log/slog"
	"net/http"

	"user-management-api/internal/core/port"
)

// NewRouter wires the handlers and middleware onto a standard library
// ServeMux (Go 1.22+ method/path patterns; no framework required).
func NewRouter(service port.UserService, tokens port.TokenService, logger *slog.Logger) http.Handler {
	handler := NewUserHandler(service)
	authenticated := AuthMiddleware(tokens)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public auth endpoints.
	mux.HandleFunc("POST /api/v1/auth/register", handler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", handler.Login)

	// Protected user CRUD endpoints.
	mux.Handle("POST /api/v1/users", authenticated(http.HandlerFunc(handler.Create)))
	mux.Handle("GET /api/v1/users", authenticated(http.HandlerFunc(handler.List)))
	mux.Handle("GET /api/v1/users/{id}", authenticated(http.HandlerFunc(handler.GetByID)))
	mux.Handle("PATCH /api/v1/users/{id}", authenticated(http.HandlerFunc(handler.Update)))
	mux.Handle("DELETE /api/v1/users/{id}", authenticated(http.HandlerFunc(handler.Delete)))

	var root http.Handler = mux
	root = LoggingMiddleware(logger)(root)
	root = RecoveryMiddleware(logger)(root)
	return root
}
