package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	grpchandler "user-management-api/internal/adapter/grpc"
	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/testutil"
)

// newTestClient starts an in-memory gRPC server (bufconn) with the auth
// interceptor installed and returns a connected client.
func newTestClient(t *testing.T, service *testutil.UserService) userv1.UserServiceClient {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(grpchandler.AuthUnaryInterceptor(testutil.TokenService{})))
	userv1.RegisterUserServiceServer(server, grpchandler.NewServer(service))

	go func() {
		if err := server.Serve(listener); err != nil {
			t.Errorf("serving: %v", err)
		}
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.DialContext(context.Background())
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dialing bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return userv1.NewUserServiceClient(conn)
}

func authContext(t *testing.T, header string) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return metadata.AppendToOutgoingContext(ctx, "authorization", header)
}

func TestCreateUserIsPublic(t *testing.T) {
	service := &testutil.UserService{
		RegisterFn: func(string, string, string) (*domain.User, error) { return testutil.User, nil },
	}
	client := newTestClient(t, service)

	resp, err := client.CreateUser(context.Background(), &userv1.CreateUserRequest{
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

func TestCreateUserMapsDomainErrors(t *testing.T) {
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

func TestGetUserRejectsBadCredentials(t *testing.T) {
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

// The gRPC interceptor and the HTTP middleware share one parser, so a
// lowercase scheme must be accepted on both transports.
func TestGetUserAcceptsLowercaseBearerScheme(t *testing.T) {
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
