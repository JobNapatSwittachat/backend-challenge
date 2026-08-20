package response

import (
	"errors"
	"log/slog"
	"net/http"

	"user-management-api/internal/core/domain"
)

// DomainError maps the core layer's sentinel errors onto HTTP status codes.
// This is the single place that decides what a domain error means over HTTP;
// unrecognized errors are logged and reported as a generic 500 so internal
// details never reach the client.
func DomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidToken):
		Error(w, http.StatusUnauthorized, err.Error())
	default:
		slog.Error("internal error", "error", err)
		Error(w, http.StatusInternalServerError, "internal server error")
	}
}
