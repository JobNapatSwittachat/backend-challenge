package user

import (
	"net/http"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/adapter/http/response"
)

// Register creates a user. It backs both the public POST /auth/register and
// the authenticated POST /users route, which differ only in who may call them.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if !decode(w, r, &req) {
		return
	}

	user, err := h.service.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		response.DomainError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, dto.NewUserResponse(user))
}
