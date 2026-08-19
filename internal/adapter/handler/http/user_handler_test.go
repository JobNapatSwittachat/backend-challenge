package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"user-management-api/internal/core/domain"
)

// serviceStub implements port.UserService for handler tests. Each field
// overrides one use case; unset fields panic if reached.
type serviceStub struct {
	registerFn func(name, email, password string) (*domain.User, error)
	loginFn    func(email, password string) (string, *domain.User, error)
	getFn      func(id string) (*domain.User, error)
	listFn     func() ([]domain.User, error)
	updateFn   func(id string, update domain.UserUpdate) (*domain.User, error)
	deleteFn   func(id string) error
}

func (s *serviceStub) Register(_ context.Context, name, email, password string) (*domain.User, error) {
	return s.registerFn(name, email, password)
}

func (s *serviceStub) Login(_ context.Context, email, password string) (string, *domain.User, error) {
	return s.loginFn(email, password)
}

func (s *serviceStub) GetByID(_ context.Context, id string) (*domain.User, error) {
	return s.getFn(id)
}

func (s *serviceStub) List(_ context.Context) ([]domain.User, error) { return s.listFn() }

func (s *serviceStub) Update(_ context.Context, id string, update domain.UserUpdate) (*domain.User, error) {
	return s.updateFn(id, update)
}

func (s *serviceStub) Delete(_ context.Context, id string) error { return s.deleteFn(id) }

func (s *serviceStub) Count(_ context.Context) (int64, error) { return 0, nil }

var testUser = &domain.User{
	ID:        "abc123",
	Name:      "Alice",
	Email:     "alice@example.com",
	CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
}

const authHeader = "Bearer token-for-user-1"

func newTestRouter(svc *serviceStub) http.Handler {
	return NewRouter(svc, stubTokens{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func doRequest(t *testing.T, handler http.Handler, method, path, body, auth string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != "" {
		request.Header.Set("Authorization", auth)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestRegisterEndpoint(t *testing.T) {
	svc := &serviceStub{
		registerFn: func(name, email, password string) (*domain.User, error) {
			if name != "Alice" || email != "alice@example.com" || password != "supersecret" {
				t.Errorf("unexpected args: %q %q %q", name, email, password)
			}
			return testUser, nil
		},
	}

	recorder := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"supersecret"}`, "")

	if recorder.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", recorder.Code, recorder.Body)
	}
	var got domain.User
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.ID != testUser.ID {
		t.Errorf("want id %q, got %q", testUser.ID, got.ID)
	}
	if strings.Contains(strings.ToLower(recorder.Body.String()), "password") {
		t.Error("response must not contain any password field")
	}
}

func TestRegisterEndpointRejectsBadJSON(t *testing.T) {
	recorder := doRequest(t, newTestRouter(&serviceStub{}), http.MethodPost, "/api/v1/auth/register", `{oops`, "")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", recorder.Code)
	}
}

func TestRegisterEndpointMapsValidationTo400(t *testing.T) {
	svc := &serviceStub{
		registerFn: func(_, _, _ string) (*domain.User, error) { return nil, domain.ErrValidation },
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/auth/register",
		`{"name":"","email":"","password":""}`, "")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", recorder.Code)
	}
}

func TestRegisterEndpointMapsDuplicateTo409(t *testing.T) {
	svc := &serviceStub{
		registerFn: func(_, _, _ string) (*domain.User, error) { return nil, domain.ErrEmailAlreadyExists },
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"supersecret"}`, "")
	if recorder.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", recorder.Code)
	}
}

func TestLoginEndpointReturnsToken(t *testing.T) {
	svc := &serviceStub{
		loginFn: func(email, password string) (string, *domain.User, error) {
			return "jwt-token", testUser, nil
		},
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/auth/login",
		`{"email":"alice@example.com","password":"supersecret"}`, "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
	var got loginResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.Token != "jwt-token" {
		t.Errorf("want token jwt-token, got %q", got.Token)
	}
}

func TestLoginEndpointMapsBadCredentialsTo401(t *testing.T) {
	svc := &serviceStub{
		loginFn: func(_, _ string) (string, *domain.User, error) {
			return "", nil, domain.ErrInvalidCredentials
		},
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/auth/login",
		`{"email":"alice@example.com","password":"wrong"}`, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", recorder.Code)
	}
}

func TestProtectedEndpointsRequireToken(t *testing.T) {
	router := newTestRouter(&serviceStub{})

	tests := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/users"},
		{http.MethodGet, "/api/v1/users"},
		{http.MethodGet, "/api/v1/users/abc123"},
		{http.MethodPatch, "/api/v1/users/abc123"},
		{http.MethodDelete, "/api/v1/users/abc123"},
	}
	for _, tc := range tests {
		recorder := doRequest(t, router, tc.method, tc.path, "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: want 401, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestGetUserEndpoint(t *testing.T) {
	svc := &serviceStub{
		getFn: func(id string) (*domain.User, error) {
			if id != "abc123" {
				t.Errorf("want id abc123, got %q", id)
			}
			return testUser, nil
		},
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/users/abc123", "", authHeader)
	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
}

func TestGetUserEndpointMapsNotFoundTo404(t *testing.T) {
	svc := &serviceStub{
		getFn: func(string) (*domain.User, error) { return nil, domain.ErrUserNotFound },
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/users/ghost", "", authHeader)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", recorder.Code)
	}
}

func TestListUsersEndpoint(t *testing.T) {
	svc := &serviceStub{
		listFn: func() ([]domain.User, error) { return []domain.User{*testUser}, nil },
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/users", "", authHeader)
	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", recorder.Code)
	}
	var got []domain.User
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(got) != 1 || got[0].ID != testUser.ID {
		t.Errorf("unexpected list response: %+v", got)
	}
}

func TestUpdateUserEndpoint(t *testing.T) {
	svc := &serviceStub{
		updateFn: func(id string, update domain.UserUpdate) (*domain.User, error) {
			if id != "abc123" {
				t.Errorf("want id abc123, got %q", id)
			}
			if update.Name == nil || *update.Name != "New Name" {
				t.Errorf("want name New Name, got %v", update.Name)
			}
			if update.Email != nil {
				t.Errorf("email should be nil, got %v", *update.Email)
			}
			return testUser, nil
		},
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodPatch, "/api/v1/users/abc123",
		`{"name":"New Name"}`, authHeader)
	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
}

func TestDeleteUserEndpoint(t *testing.T) {
	deleted := false
	svc := &serviceStub{
		deleteFn: func(id string) error {
			deleted = true
			return nil
		},
	}
	recorder := doRequest(t, newTestRouter(svc), http.MethodDelete, "/api/v1/users/abc123", "", authHeader)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", recorder.Code)
	}
	if !deleted {
		t.Error("delete was not called on the service")
	}
}

func TestHealthEndpointIsPublic(t *testing.T) {
	recorder := doRequest(t, newTestRouter(&serviceStub{}), http.MethodGet, "/healthz", "", "")
	if recorder.Code != http.StatusOK {
		t.Errorf("want 200, got %d", recorder.Code)
	}
}
