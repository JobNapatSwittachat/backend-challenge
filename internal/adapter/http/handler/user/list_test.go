package user_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

func TestListReturnsUsers(t *testing.T) {
	service := &testutil.UserService{
		ListFn: func() ([]domain.User, error) { return []domain.User{*testutil.User}, nil },
	}

	recorder := call(t, handlerFor(service).List, http.MethodGet, "/api/v1/users", "", "")

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

// Clients iterate the result, so an empty list must serialize as [] not null.
func TestListEmptySerializesAsArray(t *testing.T) {
	service := &testutil.UserService{
		ListFn: func() ([]domain.User, error) { return nil, nil },
	}
	recorder := call(t, handlerFor(service).List, http.MethodGet, "/api/v1/users", "", "")
	if body := strings.TrimSpace(recorder.Body.String()); body != "[]" {
		t.Errorf("want [], got %s", body)
	}
}
