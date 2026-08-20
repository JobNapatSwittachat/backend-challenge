// Package user holds the HTTP handlers for the user module: one file per use
// case, each decoding a DTO, calling one use case on the service port, and
// handing the result to the response package. No business logic lives here.
package user

import (
	"net/http"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/adapter/http/response"
	"user-management-api/internal/core/port"
)

// Handler serves the user endpoints.
type Handler struct {
	service port.UserService
}

func New(service port.UserService) *Handler {
	return &Handler{service: service}
}

// decode fills dst from the request body, writing a 400 and returning false
// if the payload is malformed.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := dto.Decode(r, dst); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}
