package grpc_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

func TestCreateUserIsPublic(t *testing.T) {
	service := &testutil.UserService{
		RegisterFn: func(name, email, password string) (*domain.User, error) {
			if name != "Alice" || email != "alice@example.com" || password != "supersecret" {
				t.Errorf("unexpected args: %q %q %q", name, email, password)
			}
			return testutil.User, nil
		},
	}

	// No authorization metadata: CreateUser must still succeed.
	resp, err := newTestClient(t, service).CreateUser(context.Background(), &userv1.CreateUserRequest{
		Name: "Alice", Email: "alice@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if resp.GetUser().GetId() != testutil.User.ID {
		t.Errorf("want id %q, got %q", testutil.User.ID, resp.GetUser().GetId())
	}
	if resp.GetUser().GetCreatedAt().AsTime() != testutil.User.CreatedAt {
		t.Errorf("want created_at %v, got %v", testutil.User.CreatedAt, resp.GetUser().GetCreatedAt().AsTime())
	}
}

func TestCreateUserErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"duplicate email", domain.ErrEmailAlreadyExists, codes.AlreadyExists},
		{"validation", domain.ErrValidation, codes.InvalidArgument},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &testutil.UserService{
				RegisterFn: func(string, string, string) (*domain.User, error) { return nil, tc.err },
			}
			_, err := newTestClient(t, service).CreateUser(context.Background(), &userv1.CreateUserRequest{
				Name: "Bob", Email: "bob@example.com", Password: "supersecret",
			})
			if status.Code(err) != tc.wantCode {
				t.Errorf("want %v, got %v", tc.wantCode, err)
			}
		})
	}
}
