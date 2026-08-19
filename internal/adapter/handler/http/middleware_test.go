package http

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"user-management-api/internal/core/domain"
)

type stubTokens struct{}

func (stubTokens) Generate(userID string) (string, error) { return "token-for-" + userID, nil }

func (stubTokens) Verify(token string) (string, error) {
	id, ok := strings.CutPrefix(token, "token-for-")
	if !ok {
		return "", domain.ErrInvalidToken
	}
	return id, nil
}

func TestAuthMiddlewareRejectsMissingHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached without a token")
	})
	recorder := httptest.NewRecorder()

	AuthMiddleware(stubTokens{})(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", recorder.Code)
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached with an invalid token")
	})
	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	request.Header.Set("Authorization", "Bearer bogus")
	recorder := httptest.NewRecorder()

	AuthMiddleware(stubTokens{})(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", recorder.Code)
	}
}

func TestAuthMiddlewarePassesValidTokenAndSetsUserID(t *testing.T) {
	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = UserIDFromContext(r.Context())
	})
	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	request.Header.Set("Authorization", "Bearer token-for-user-42")
	recorder := httptest.NewRecorder()

	AuthMiddleware(stubTokens{})(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("want 200, got %d", recorder.Code)
	}
	if gotUserID != "user-42" {
		t.Errorf("want user-42 in context, got %q", gotUserID)
	}
}

func TestLoggingMiddlewareLogsMethodPathStatusDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	recorder := httptest.NewRecorder()
	LoggingMiddleware(logger)(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/some/path", nil))

	logged := buf.String()
	for _, want := range []string{"method=DELETE", "path=/some/path", "status=418", "duration="} {
		if !strings.Contains(logged, want) {
			t.Errorf("log line missing %q: %s", want, logged)
		}
	}
}

func TestRecoveryMiddlewareTurnsPanicInto500(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	recorder := httptest.NewRecorder()

	RecoveryMiddleware(logger)(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", recorder.Code)
	}
}
