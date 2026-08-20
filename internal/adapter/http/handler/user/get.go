package user

import (
	"net/http"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/adapter/http/response"
)

// Get returns the user named by the {id} path value.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		response.DomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewUserResponse(user))
}
