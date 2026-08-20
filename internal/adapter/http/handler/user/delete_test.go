package user_test

import (
	"net/http"
	"testing"

	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

func TestDeleteReturns204(t *testing.T) {
	var gotID string
	service := &testutil.UserService{
		DeleteFn: func(id string) error {
			gotID = id
			return nil
		},
	}

	recorder := call(t, handlerFor(service).Delete, http.MethodDelete, "/api/v1/users/abc123", "", "abc123")

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", recorder.Code)
	}
	if gotID != "abc123" {
		t.Errorf("want delete called with abc123, got %q", gotID)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("204 must have an empty body, got %s", recorder.Body)
	}
}

func TestDeleteMapsNotFoundTo404(t *testing.T) {
	service := &testutil.UserService{
		DeleteFn: func(string) error { return domain.ErrUserNotFound },
	}
	recorder := call(t, handlerFor(service).Delete, http.MethodDelete, "/api/v1/users/ghost", "", "ghost")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", recorder.Code)
	}
}
