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

func TestProtectedRPCRejectsBadCredentials(t *testing.T) {
	service := &testutil.UserService{
		GetByIDFn: func(string) (*domain.User, error) {
			t.Error("service must not be reached without a valid token")
			return nil, nil
		},
	}
	client := newTestClient(t, service)

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{"no metadata", context.Background()},
		{"invalid token", authContext(t, "Bearer bogus")},
		{"missing bearer scheme", authContext(t, "token-for-user-1")},
		{"wrong scheme", authContext(t, "Basic token-for-user-1")},
		{"empty credential", authContext(t, "Bearer ")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.GetUser(tc.ctx, &userv1.GetUserRequest{Id: testutil.User.ID})
			if status.Code(err) != codes.Unauthenticated {
				t.Errorf("want Unauthenticated, got %v", err)
			}
		})
	}
}

// The interceptor and the HTTP middleware share one parser, so a lowercase
// scheme must be accepted on both transports.
func TestProtectedRPCAcceptsLowercaseBearerScheme(t *testing.T) {
	service := &testutil.UserService{
		GetByIDFn: func(string) (*domain.User, error) { return testutil.User, nil },
	}

	_, err := newTestClient(t, service).GetUser(
		authContext(t, "bearer token-for-user-1"),
		&userv1.GetUserRequest{Id: testutil.User.ID},
	)
	if err != nil {
		t.Errorf("lowercase bearer scheme should be accepted: %v", err)
	}
}
