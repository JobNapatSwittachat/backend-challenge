// Package router maps HTTP routes onto handlers and composes the middleware
// chain. It is the only place that knows the URL layout of the API.
package router

import (
	"log/slog"
	"net/http"

	"user-management-api/internal/adapter/http/handler"
	"user-management-api/internal/adapter/http/middleware"
	"user-management-api/internal/core/port"
)

// New builds the API handler on a standard library ServeMux, using the Go
// 1.22+ method/path patterns rather than a third-party router.
func New(service port.UserService, tokens port.TokenService, logger *slog.Logger) http.Handler {
	users := handler.NewUserHandler(service)
	authenticated := middleware.Auth(tokens)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", users.Health)

	// Public: a client needs these before it can hold a token.
	mux.HandleFunc("POST /api/v1/auth/register", users.Register)
	mux.HandleFunc("POST /api/v1/auth/login", users.Login)

	// Protected user CRUD. POST /users is the authenticated twin of register
	// and shares its use case.
	mux.Handle("POST /api/v1/users", authenticated(http.HandlerFunc(users.Register)))
	mux.Handle("GET /api/v1/users", authenticated(http.HandlerFunc(users.List)))
	mux.Handle("GET /api/v1/users/{id}", authenticated(http.HandlerFunc(users.GetByID)))
	mux.Handle("PATCH /api/v1/users/{id}", authenticated(http.HandlerFunc(users.Update)))
	mux.Handle("DELETE /api/v1/users/{id}", authenticated(http.HandlerFunc(users.Delete)))

	// Outermost middleware runs first: recovery wraps logging so a panic is
	// still logged as a completed 500 request.
	var root http.Handler = mux
	root = middleware.Logging(logger)(root)
	root = middleware.Recovery(logger)(root)
	return root
}
