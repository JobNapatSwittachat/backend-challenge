package http

import (
	"encoding/json"
	"net/http"

	"user-management-api/internal/core/domain"
	"user-management-api/internal/core/port"
)

// UserHandler exposes the user use cases over HTTP.
type UserHandler struct {
	service port.UserService
}

func NewUserHandler(service port.UserService) *UserHandler {
	return &UserHandler{service: service}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

type updateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// Register handles POST /api/v1/auth/register (public).
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeBody(w, r, &req) {
		return
	}
	user, err := h.service.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// Login handles POST /api/v1/auth/login (public). Returns a JWT.
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeBody(w, r, &req) {
		return
	}
	token, user, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: user})
}

// Create handles POST /api/v1/users (protected). Functionally identical to
// Register but requires authentication, matching the "Create a new user"
// operation in the challenge.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	h.Register(w, r)
}

// GetByID handles GET /api/v1/users/{id} (protected).
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// List handles GET /api/v1/users (protected).
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// Update handles PATCH /api/v1/users/{id} (protected).
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateUserRequest
	if !decodeBody(w, r, &req) {
		return
	}
	user, err := h.service.Update(r.Context(), r.PathValue("id"), domain.UserUpdate{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// Delete handles DELETE /api/v1/users/{id} (protected).
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
