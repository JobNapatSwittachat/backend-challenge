package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/adapter/http/handler"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

// call invokes a handler directly. pathID, when non-empty, populates the
// {id} path value the router would normally set.
func call(t *testing.T, h http.HandlerFunc, method, target, body, pathID string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if pathID != "" {
		request.SetPathValue("id", pathID)
	}
	recorder := httptest.NewRecorder()
	h(recorder, request)
	return recorder
}

func TestRegister(t *testing.T) {
	service := &testutil.UserService{
		RegisterFn: func(name, email, password string) (*domain.User, error) {
			if name != "Alice" || email != "alice@example.com" || password != "supersecret" {
				t.Errorf("unexpected args: %q %q %q", name, email, password)
			}
			return testutil.User, nil
		},
	}

	recorder := call(t, handler.NewUserHandler(service).Register, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"supersecret"}`, "")

	if recorder.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", recorder.Code, recorder.Body)
	}
	var got dto.UserResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.ID != testutil.User.ID {
		t.Errorf("want id %q, got %q", testutil.User.ID, got.ID)
	}
	if strings.Contains(strings.ToLower(recorder.Body.String()), "password") {
		t.Errorf("response must not mention any password field: %s", recorder.Body)
	}
}

func TestRegisterRejectsMalformedJSON(t *testing.T) {
	recorder := call(t, handler.NewUserHandler(&testutil.UserService{}).Register,
		http.MethodPost, "/api/v1/auth/register", `{oops`, "")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", recorder.Code)
	}
}

func TestRegisterRejectsUnknownFields(t *testing.T) {
	recorder := call(t, handler.NewUserHandler(&testutil.UserService{}).Register,
		http.MethodPost, "/api/v1/auth/register",
		`{"name":"Alice","email":"a@b.com","password":"supersecret","admin":true}`, "")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("want 400 for unknown field, got %d", recorder.Code)
	}
}

func TestRegisterErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"validation", domain.ErrValidation, http.StatusBadRequest},
		{"duplicate email", domain.ErrEmailAlreadyExists, http.StatusConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &testutil.UserService{
				RegisterFn: func(string, string, string) (*domain.User, error) { return nil, tc.err },
			}
			recorder := call(t, handler.NewUserHandler(service).Register, http.MethodPost, "/api/v1/auth/register",
				`{"name":"Alice","email":"alice@example.com","password":"supersecret"}`, "")
			if recorder.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, recorder.Code)
			}
		})
	}
}

func TestLoginReturnsToken(t *testing.T) {
	service := &testutil.UserService{
		LoginFn: func(string, string) (string, *domain.User, error) {
			return "jwt-token", testutil.User, nil
		},
	}
	recorder := call(t, handler.NewUserHandler(service).Login, http.MethodPost, "/api/v1/auth/login",
		`{"email":"alice@example.com","password":"supersecret"}`, "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
	var got dto.LoginResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.Token != "jwt-token" {
		t.Errorf("want token jwt-token, got %q", got.Token)
	}
	if got.User.Email != testutil.User.Email {
		t.Errorf("want user %q, got %q", testutil.User.Email, got.User.Email)
	}
}

func TestLoginMapsBadCredentialsTo401(t *testing.T) {
	service := &testutil.UserService{
		LoginFn: func(string, string) (string, *domain.User, error) {
			return "", nil, domain.ErrInvalidCredentials
		},
	}
	recorder := call(t, handler.NewUserHandler(service).Login, http.MethodPost, "/api/v1/auth/login",
		`{"email":"alice@example.com","password":"wrong"}`, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", recorder.Code)
	}
}

func TestGetByID(t *testing.T) {
	service := &testutil.UserService{
		GetByIDFn: func(id string) (*domain.User, error) {
			if id != testutil.User.ID {
				t.Errorf("want id %q, got %q", testutil.User.ID, id)
			}
			return testutil.User, nil
		},
	}
	recorder := call(t, handler.NewUserHandler(service).GetByID, http.MethodGet, "/api/v1/users/abc123", "", testutil.User.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
}

func TestGetByIDMapsNotFoundTo404(t *testing.T) {
	service := &testutil.UserService{
		GetByIDFn: func(string) (*domain.User, error) { return nil, domain.ErrUserNotFound },
	}
	recorder := call(t, handler.NewUserHandler(service).GetByID, http.MethodGet, "/api/v1/users/ghost", "", "ghost")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", recorder.Code)
	}
}

func TestList(t *testing.T) {
	service := &testutil.UserService{
		ListFn: func() ([]domain.User, error) { return []domain.User{*testutil.User}, nil },
	}
	recorder := call(t, handler.NewUserHandler(service).List, http.MethodGet, "/api/v1/users", "", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", recorder.Code)
	}
	var got []dto.UserResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(got) != 1 || got[0].ID != testutil.User.ID {
		t.Errorf("unexpected list response: %+v", got)
	}
}

func TestListEmptySerializesAsArray(t *testing.T) {
	service := &testutil.UserService{
		ListFn: func() ([]domain.User, error) { return nil, nil },
	}
	recorder := call(t, handler.NewUserHandler(service).List, http.MethodGet, "/api/v1/users", "", "")
	if body := strings.TrimSpace(recorder.Body.String()); body != "[]" {
		t.Errorf("want [] for an empty list, got %s", body)
	}
}

func TestUpdate(t *testing.T) {
	service := &testutil.UserService{
		UpdateFn: func(id string, update domain.UserUpdate) (*domain.User, error) {
			if id != testutil.User.ID {
				t.Errorf("want id %q, got %q", testutil.User.ID, id)
			}
			if update.Name == nil || *update.Name != "New Name" {
				t.Errorf("want name New Name, got %v", update.Name)
			}
			if update.Email != nil {
				t.Errorf("omitted email should stay nil, got %q", *update.Email)
			}
			return testutil.User, nil
		},
	}
	recorder := call(t, handler.NewUserHandler(service).Update, http.MethodPatch, "/api/v1/users/abc123",
		`{"name":"New Name"}`, testutil.User.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
}

func TestUpdateMapsDuplicateEmailTo409(t *testing.T) {
	service := &testutil.UserService{
		UpdateFn: func(string, domain.UserUpdate) (*domain.User, error) {
			return nil, domain.ErrEmailAlreadyExists
		},
	}
	recorder := call(t, handler.NewUserHandler(service).Update, http.MethodPatch, "/api/v1/users/abc123",
		`{"email":"taken@example.com"}`, testutil.User.ID)
	if recorder.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", recorder.Code)
	}
}

func TestDelete(t *testing.T) {
	deleted := false
	service := &testutil.UserService{
		DeleteFn: func(string) error {
			deleted = true
			return nil
		},
	}
	recorder := call(t, handler.NewUserHandler(service).Delete, http.MethodDelete, "/api/v1/users/abc123", "", testutil.User.ID)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", recorder.Code)
	}
	if !deleted {
		t.Error("delete was not called on the service")
	}
}

func TestDeleteMapsNotFoundTo404(t *testing.T) {
	service := &testutil.UserService{
		DeleteFn: func(string) error { return domain.ErrUserNotFound },
	}
	recorder := call(t, handler.NewUserHandler(service).Delete, http.MethodDelete, "/api/v1/users/ghost", "", "ghost")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", recorder.Code)
	}
}
