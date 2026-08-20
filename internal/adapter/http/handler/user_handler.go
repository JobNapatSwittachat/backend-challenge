// Package handler implements the HTTP request handlers. Each handler decodes
// a DTO, calls one use case on the service port, and hands the result to the
// response package — no business logic lives here.
package handler

import (
	"net/http"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/adapter/http/response"
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

// decode fills dst from the request body, writing a 400 and returning false
// if the payload is malformed.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := dto.Decode(r, dst); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// Register handles user creation. It backs both the public
// POST /auth/register and the authenticated POST /users route.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
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

// Login verifies credentials and returns a JWT.
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if !decode(w, r, &req) {
		return
	}
	token, user, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.DomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.LoginResponse{
		Token: token,
		User:  dto.NewUserResponse(user),
	})
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		response.DomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewUserResponse(user))
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		response.DomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewUserResponses(users))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
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

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); err != nil {
		response.DomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Health reports service liveness.
func (h *UserHandler) Health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
