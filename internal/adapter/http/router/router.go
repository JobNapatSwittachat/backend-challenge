// Package router maps HTTP routes onto handlers and composes the middleware
// chain. It is the only place that knows the URL layout of the API.
package router

import (
	"log/slog"
	"net/http"

	"user-management-api/internal/adapter/http/handler/health"
	"user-management-api/internal/adapter/http/handler/user"
	"user-management-api/internal/adapter/http/middleware"
	"user-management-api/internal/core/port"
)

// New builds the API handler on a standard library ServeMux, using the Go
// 1.22+ method/path patterns rather than a third-party router.
func New(service port.UserService, tokens port.TokenService, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	registerHealthRoutes(mux)
	registerUserRoutes(mux, user.New(service), middleware.Auth(tokens))

	// The outermost middleware runs first, so recovery wraps logging and a
	// panic is still logged as a completed 500 request.
	var root http.Handler = mux
	root = middleware.Logging(logger)(root)
	root = middleware.Recovery(logger)(root)
	return root
}

func registerHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", health.Check)
}

// registerUserRoutes wires the user module. Auth is applied per route rather
// than to the whole mux, so which endpoints are public is visible here.
func registerUserRoutes(mux *http.ServeMux, users *user.Handler, auth func(http.Handler) http.Handler) {
	// Public: a client needs these before it can hold a token.
	mux.HandleFunc("POST /api/v1/auth/register", users.Register)
	mux.HandleFunc("POST /api/v1/auth/login", users.Login)

	// Protected CRUD. POST /users is the authenticated twin of register and
	// shares its use case.
	mux.Handle("POST /api/v1/users", auth(http.HandlerFunc(users.Register)))
	mux.Handle("GET /api/v1/users", auth(http.HandlerFunc(users.List)))
	mux.Handle("GET /api/v1/users/{id}", auth(http.HandlerFunc(users.Get)))
	mux.Handle("PATCH /api/v1/users/{id}", auth(http.HandlerFunc(users.Update)))
	mux.Handle("DELETE /api/v1/users/{id}", auth(http.HandlerFunc(users.Delete)))
}
