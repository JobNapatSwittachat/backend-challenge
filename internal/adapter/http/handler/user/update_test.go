package user_test

import (
	"net/http"
	"testing"

	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

func TestUpdateSendsOnlyProvidedFields(t *testing.T) {
	service := &testutil.UserService{
		UpdateFn: func(id string, update domain.UserUpdate) (*domain.User, error) {
			if id != "abc123" {
				t.Errorf("want id abc123, got %q", id)
			}
			if update.Name == nil || *update.Name != "New Name" {
				t.Errorf("want name New Name, got %v", update.Name)
			}
			if update.Email != nil {
				t.Errorf("omitted email must stay nil, got %q", *update.Email)
			}
			return testutil.User, nil
		},
	}

	recorder := call(t, handlerFor(service).Update, http.MethodPatch, "/api/v1/users/abc123",
		`{"name":"New Name"}`, "abc123")

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", recorder.Code, recorder.Body)
	}
}

func TestUpdateErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"duplicate email", domain.ErrEmailAlreadyExists, http.StatusConflict},
		{"unknown user", domain.ErrUserNotFound, http.StatusNotFound},
		{"validation", domain.ErrValidation, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &testutil.UserService{
				UpdateFn: func(string, domain.UserUpdate) (*domain.User, error) { return nil, tc.err },
			}
			recorder := call(t, handlerFor(service).Update, http.MethodPatch, "/api/v1/users/abc123",
				`{"email":"taken@example.com"}`, "abc123")
			if recorder.Code != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, recorder.Code)
			}
		})
	}
}

func TestUpdateRejectsMalformedBody(t *testing.T) {
	recorder := call(t, handlerFor(&testutil.UserService{}).Update, http.MethodPatch, "/api/v1/users/abc123",
		`{"name":}`, "abc123")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", recorder.Code)
	}
}
