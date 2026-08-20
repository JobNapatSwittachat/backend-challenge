// Package response writes HTTP responses, so every endpoint reports success
// and failure in the same shape.
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"user-management-api/internal/adapter/http/dto"
)

// JSON writes status and body. A nil body writes no payload.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The status line is already sent, so this can only be logged.
		slog.Error("encoding response", "error", err)
	}
}

// Error writes a status with the standard error body.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, dto.ErrorResponse{Error: message})
}
