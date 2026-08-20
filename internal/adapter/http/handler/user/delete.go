package user

import (
	"net/http"

	"user-management-api/internal/adapter/http/response"
)

// Delete removes the user named by the {id} path value.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); err != nil {
		response.DomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
