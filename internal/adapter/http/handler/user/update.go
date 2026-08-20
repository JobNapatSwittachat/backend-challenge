package user

import (
	"net/http"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/adapter/http/response"
	"user-management-api/internal/core/domain"
)

// Update applies a partial change to the user named by the {id} path value.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateUserRequest
	if !decode(w, r, &req) {
		return
	}

	user, err := h.service.Update(r.Context(), r.PathValue("id"), domain.UserUpdate{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		response.DomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewUserResponse(user))
}
