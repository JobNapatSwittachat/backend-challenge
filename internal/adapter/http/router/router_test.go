package router_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"user-management-api/internal/adapter/http/router"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

// newRouter builds the real route table over fakes, so these tests cover
// wiring — method/path matching, path variables, and which routes the auth
// middleware protects.
func newRouter(service *testutil.UserService) http.Handler {
	return router.New(service, testutil.TokenService{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func request(t *testing.T, handler http.Handler, method, path, body, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func TestProtectedRoutesRequireToken(t *testing.T) {
	// Every use case panics if called: reaching the service at all means the
	// middleware failed to block the request.
	handler := newRouter(&testutil.UserService{})

	tests := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/users"},
		{http.MethodGet, "/api/v1/users"},
		{http.MethodGet, "/api/v1/users/abc123"},
		{http.MethodPatch, "/api/v1/users/abc123"},
		{http.MethodDelete, "/api/v1/users/abc123"},
	}
	for _, tc := range tests {
		recorder := request(t, handler, tc.method, tc.path, "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: want 401, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestPublicRoutesNeedNoToken(t *testing.T) {
	service := &testutil.UserService{
		RegisterFn: func(string, string, string) (*domain.User, error) { return testutil.User, nil },
		LoginFn: func(string, string) (string, *domain.User, error) {
			return "jwt-token", testutil.User, nil
		},
	}
	handler := newRouter(service)

	tests := []struct {
		method, path, body string
		wantStatus         int
	}{
		{http.MethodGet, "/healthz", "", http.StatusOK},
		{http.MethodPost, "/api/v1/auth/register", `{"name":"Alice","email":"a@b.com","password":"supersecret"}`, http.StatusCreated},
		{http.MethodPost, "/api/v1/auth/login", `{"email":"a@b.com","password":"supersecret"}`, http.StatusOK},
	}
	for _, tc := range tests {
		recorder := request(t, handler, tc.method, tc.path, tc.body, "")
		if recorder.Code != tc.wantStatus {
			t.Errorf("%s %s: want %d, got %d (%s)", tc.method, tc.path, tc.wantStatus, recorder.Code, recorder.Body)
		}
	}
}

func TestAuthenticatedRouteReachesHandlerWithPathValue(t *testing.T) {
	var gotID string
	service := &testutil.UserService{
		GetByIDFn: func(id string) (*domain.User, error) {
			gotID = id
			return testutil.User, nil
		},
	}

	recorder := request(t, newRouter(service), http.MethodGet, "/api/v1/users/abc123", "", testutil.AuthHeader)

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
	if gotID != "abc123" {
		t.Errorf("want path value abc123 to reach the handler, got %q", gotID)
	}
}

func TestUnknownRouteAndMethodMismatch(t *testing.T) {
	handler := newRouter(&testutil.UserService{})

	if recorder := request(t, handler, http.MethodGet, "/nope", "", ""); recorder.Code != http.StatusNotFound {
		t.Errorf("unknown path: want 404, got %d", recorder.Code)
	}
	if recorder := request(t, handler, http.MethodPut, "/api/v1/users/abc123", "", testutil.AuthHeader); recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("unregistered method: want 405, got %d", recorder.Code)
	}
}

func TestPanicInHandlerBecomes500(t *testing.T) {
	// ListFn is nil, so the handler panics — the recovery middleware must
	// turn that into a 500 rather than crashing the server.
	recorder := request(t, newRouter(&testutil.UserService{}), http.MethodGet, "/api/v1/users", "", testutil.AuthHeader)
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("want 500 from recovery middleware, got %d", recorder.Code)
	}
}
