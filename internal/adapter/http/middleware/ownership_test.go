package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"user-management-api/internal/adapter/http/middleware"
	"user-management-api/internal/testutil"
)

// requestAs builds a request for /users/{pathID} carrying the given token,
// run through Auth so the user ID is in context, then RequireSelf.
func requestAs(t *testing.T, token, pathID string, next http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+pathID, nil)
	request.SetPathValue("id", pathID)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	middleware.Auth(testutil.TokenService{})(middleware.RequireSelf(next)).ServeHTTP(recorder, request)
	return recorder
}

func TestRequireSelfAllowsOwnAccount(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })

	recorder := requestAs(t, "token-for-user-1", "user-1", next)

	if recorder.Code != http.StatusOK {
		t.Errorf("want 200, got %d", recorder.Code)
	}
	if !reached {
		t.Error("handler should run when the id matches the authenticated user")
	}
}

func TestRequireSelfBlocksAnotherAccount(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler must not run for someone else's account")
	})

	recorder := requestAs(t, "token-for-user-1", "user-2", next)

	if recorder.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", recorder.Code)
	}
}

// Without Auth in front there is no user ID, which must fail closed rather
// than fall through to the handler.
func TestRequireSelfWithoutAuthContext(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler must not run without an authenticated user")
	})
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/users/user-1", nil)
	request.SetPathValue("id", "user-1")
	recorder := httptest.NewRecorder()

	middleware.RequireSelf(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", recorder.Code)
	}
}
