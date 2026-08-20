// Package response writes HTTP responses and translates core-layer errors
// into status codes, so every endpoint reports success and failure the same
// way.
package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/core/domain"
)

// JSON writes status and body. A nil body writes no payload.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		if err := json.NewEncoder(w).Encode(body); err != nil {
			slog.Error("encoding response", "error", err)
		}
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, dto.ErrorResponse{Error: message})
}

// DomainError maps the core layer's sentinel errors onto HTTP status codes.
// Unrecognized errors are logged and reported as a generic 500 so internal
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
