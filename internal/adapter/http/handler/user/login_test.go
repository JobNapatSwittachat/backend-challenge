package user_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"user-management-api/internal/adapter/http/dto"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

const loginBody = `{"email":"alice@example.com","password":"supersecret"}`

func TestLoginReturnsTokenAndUser(t *testing.T) {
	service := &testutil.UserService{
		LoginFn: func(email, password string) (string, *domain.User, error) {
			if email != "alice@example.com" || password != "supersecret" {
				t.Errorf("unexpected credentials: %q %q", email, password)
			}
			return "jwt-token", testutil.User, nil
		},
	}

	recorder := call(t, handlerFor(service).Login, http.MethodPost, "/api/v1/auth/login", loginBody, "")

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
	recorder := call(t, handlerFor(service).Login, http.MethodPost, "/api/v1/auth/login", loginBody, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", recorder.Code)
	}
}

func TestLoginRejectsMalformedBody(t *testing.T) {
	recorder := call(t, handlerFor(&testutil.UserService{}).Login, http.MethodPost, "/api/v1/auth/login", `{`, "")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", recorder.Code)
	}
}
