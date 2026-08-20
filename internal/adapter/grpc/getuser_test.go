package grpc_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

func TestGetUserWithToken(t *testing.T) {
	service := &testutil.UserService{
		GetByIDFn: func(id string) (*domain.User, error) {
			if id != testutil.User.ID {
				return nil, domain.ErrUserNotFound
			}
			return testutil.User, nil
		},
	}

	resp, err := newTestClient(t, service).GetUser(
		authContext(t, testutil.AuthHeader),
		&userv1.GetUserRequest{Id: testutil.User.ID},
	)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if resp.GetUser().GetEmail() != testutil.User.Email {
		t.Errorf("want email %q, got %q", testutil.User.Email, resp.GetUser().GetEmail())
	}
}

func TestGetUserNotFound(t *testing.T) {
	service := &testutil.UserService{
		GetByIDFn: func(string) (*domain.User, error) { return nil, domain.ErrUserNotFound },
	}
	_, err := newTestClient(t, service).GetUser(
		authContext(t, testutil.AuthHeader),
		&userv1.GetUserRequest{Id: "ghost"},
	)
	if status.Code(err) != codes.NotFound {
		t.Errorf("want NotFound, got %v", err)
	}
}
