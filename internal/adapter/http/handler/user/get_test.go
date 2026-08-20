package user_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

func TestGetReadsIDFromPath(t *testing.T) {
	var gotID string
	service := &testutil.UserService{
		GetByIDFn: func(id string) (*domain.User, error) {
			gotID = id
			return testutil.User, nil
		},
	}

	recorder := call(t, handlerFor(service).Get, http.MethodGet, "/api/v1/users/abc123", "", "abc123")

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
	if gotID != "abc123" {
		t.Errorf("want id abc123 passed to the service, got %q", gotID)
	}
	var got dto.UserResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.Email != testutil.User.Email {
		t.Errorf("want email %q, got %q", testutil.User.Email, got.Email)
	}
}

func TestGetMapsNotFoundTo404(t *testing.T) {
	service := &testutil.UserService{
		GetByIDFn: func(string) (*domain.User, error) { return nil, domain.ErrUserNotFound },
	}
	recorder := call(t, handlerFor(service).Get, http.MethodGet, "/api/v1/users/ghost", "", "ghost")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", recorder.Code)
	}
}
