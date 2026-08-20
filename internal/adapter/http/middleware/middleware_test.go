package middleware_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"user-management-api/internal/adapter/http/middleware"
	"user-management-api/internal/testutil"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{"canonical scheme", "Bearer abc.def.ghi", "abc.def.ghi", true},
		{"lowercase scheme is accepted per RFC 7235", "bearer abc.def.ghi", "abc.def.ghi", true},
		{"uppercase scheme is accepted", "BEARER abc.def.ghi", "abc.def.ghi", true},
		{"surrounding spaces are trimmed", "Bearer   abc.def.ghi  ", "abc.def.ghi", true},
		{"missing header", "", "", false},
		{"wrong scheme", "Basic abc.def.ghi", "", false},
		{"scheme with no credential", "Bearer ", "", false},
		{"bare token without scheme", "abc.def.ghi", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, ok := middleware.BearerToken(tc.header)
			if ok != tc.wantOK || token != tc.wantToken {
				t.Errorf("BearerToken(%q) = (%q, %v), want (%q, %v)", tc.header, token, ok, tc.wantToken, tc.wantOK)
			}
		})
	}
}

func TestAuthRejectsMissingHeader(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be reached without a token")
	})
	recorder := httptest.NewRecorder()

	middleware.Auth(testutil.TokenService{})(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", recorder.Code)
	}
}

func TestAuthRejectsInvalidToken(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be reached with an invalid token")
	})
	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	request.Header.Set("Authorization", "Bearer bogus")
	recorder := httptest.NewRecorder()

	middleware.Auth(testutil.TokenService{})(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", recorder.Code)
	}
}

func TestAuthPassesValidTokenAndSetsUserID(t *testing.T) {
	var gotUserID string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUserID, _ = middleware.UserIDFromContext(r.Context())
	})
	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	request.Header.Set("Authorization", testutil.AuthHeader)
	recorder := httptest.NewRecorder()

	middleware.Auth(testutil.TokenService{})(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("want 200, got %d", recorder.Code)
	}
	if gotUserID != "user-1" {
		t.Errorf("want user-1 in context, got %q", gotUserID)
	}
}

func TestUserIDFromContextWithoutAuth(t *testing.T) {
	if _, ok := middleware.UserIDFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context()); ok {
		t.Error("want ok=false for a request that never passed through Auth")
	}
}

func TestLoggingLogsMethodPathStatusDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	recorder := httptest.NewRecorder()
	middleware.Logging(logger)(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/some/path", nil))

	logged := buf.String()
	for _, want := range []string{"method=DELETE", "path=/some/path", "status=418", "duration="} {
		if !strings.Contains(logged, want) {
			t.Errorf("log line missing %q: %s", want, logged)
		}
	}
}

func TestRecoveryTurnsPanicInto500(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	recorder := httptest.NewRecorder()

	middleware.Recovery(discardLogger())(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", recorder.Code)
	}
}
