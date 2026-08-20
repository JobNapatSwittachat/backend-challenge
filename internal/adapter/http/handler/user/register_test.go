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

const registerBody = `{"name":"Alice","email":"alice@example.com","password":"supersecret"}`

func TestRegisterPassesFieldsThroughAndReturns201(t *testing.T) {
	service := &testutil.UserService{
		RegisterFn: func(name, email, password string) (*domain.User, error) {
			if name != "Alice" || email != "alice@example.com" || password != "supersecret" {
				t.Errorf("unexpected args: %q %q %q", name, email, password)
			}
			return testutil.User, nil
		},
	}

	recorder := call(t, handlerFor(service).Register, http.MethodPost, "/api/v1/auth/register", registerBody, "")

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
}

func TestRegisterNeverEchoesPassword(t *testing.T) {
	service := &testutil.UserService{
		RegisterFn: func(string, string, string) (*domain.User, error) { return testutil.User, nil },
	}
	recorder := call(t, handlerFor(service).Register, http.MethodPost, "/api/v1/auth/register", registerBody, "")
	if strings.Contains(strings.ToLower(recorder.Body.String()), "password") {
		t.Errorf("response must not mention any password field: %s", recorder.Body)
	}
}

func TestRegisterRejectsMalformedBody(t *testing.T) {
	tests := []struct{ name, body string }{
		{"invalid JSON", `{oops`},
		{"unknown field", `{"name":"A","email":"a@b.com","password":"supersecret","admin":true}`},
		{"empty body", ``},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := call(t, handlerFor(&testutil.UserService{}).Register,
				http.MethodPost, "/api/v1/auth/register", tc.body, "")
			if recorder.Code != http.StatusBadRequest {
				t.Errorf("want 400, got %d", recorder.Code)
			}
		})
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
			recorder := call(t, handlerFor(service).Register, http.MethodPost, "/api/v1/auth/register", registerBody, "")
			if recorder.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, recorder.Code)
			}
		})
	}
}
