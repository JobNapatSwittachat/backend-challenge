package grpc

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"user-management-api/internal/adapter/handler/grpc/pb"
	"user-management-api/internal/core/domain"
)

// stubService implements port.UserService for gRPC server tests.
type stubService struct{}

var stubUser = &domain.User{
	ID:        "abc123",
	Name:      "Alice",
	Email:     "alice@example.com",
	CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
}

func (stubService) Register(_ context.Context, name, email, password string) (*domain.User, error) {
	if email == "taken@example.com" {
		return nil, domain.ErrEmailAlreadyExists
	}
	if password == "" {
		return nil, domain.ErrValidation
	}
	return stubUser, nil
}

func (stubService) Login(context.Context, string, string) (string, *domain.User, error) {
	panic("unused")
}

func (stubService) GetByID(_ context.Context, id string) (*domain.User, error) {
	if id != stubUser.ID {
		return nil, domain.ErrUserNotFound
	}
	return stubUser, nil
}

func (stubService) List(context.Context) ([]domain.User, error) { panic("unused") }

func (stubService) Update(context.Context, string, domain.UserUpdate) (*domain.User, error) {
	panic("unused")
}

func (stubService) Delete(context.Context, string) error { panic("unused") }

func (stubService) Count(context.Context) (int64, error) { panic("unused") }

type stubTokens struct{}

func (stubTokens) Generate(userID string) (string, error) { return "token-for-" + userID, nil }

func (stubTokens) Verify(token string) (string, error) {
	id, ok := strings.CutPrefix(token, "token-for-")
	if !ok {
		return "", domain.ErrInvalidToken
	}
	return id, nil
}

// newTestClient starts an in-memory gRPC server (bufconn) with the auth
// interceptor installed and returns a connected client.
func newTestClient(t *testing.T) pb.UserServiceClient {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(AuthUnaryInterceptor(stubTokens{})))
	pb.RegisterUserServiceServer(server, NewServer(stubService{}))

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

	return pb.NewUserServiceClient(conn)
}

func authContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer token-for-user-1")
}

func TestCreateUserIsPublic(t *testing.T) {
	client := newTestClient(t)

	resp, err := client.CreateUser(context.Background(), &pb.CreateUserRequest{
		Name: "Alice", Email: "alice@example.com", Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if resp.GetUser().GetId() != stubUser.ID {
		t.Errorf("want id %q, got %q", stubUser.ID, resp.GetUser().GetId())
	}
	if resp.GetUser().GetCreatedAt().AsTime() != stubUser.CreatedAt {
		t.Errorf("want created_at %v, got %v", stubUser.CreatedAt, resp.GetUser().GetCreatedAt().AsTime())
	}
}

func TestCreateUserMapsDomainErrors(t *testing.T) {
	client := newTestClient(t)

	_, err := client.CreateUser(context.Background(), &pb.CreateUserRequest{
		Name: "Bob", Email: "taken@example.com", Password: "supersecret",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("want AlreadyExists, got %v", err)
	}

	_, err = client.CreateUser(context.Background(), &pb.CreateUserRequest{Name: "Bob"})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestGetUserRequiresToken(t *testing.T) {
	client := newTestClient(t)

	_, err := client.GetUser(context.Background(), &pb.GetUserRequest{Id: stubUser.ID})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated without metadata, got %v", err)
	}

	badCtx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer bogus")
	_, err = client.GetUser(badCtx, &pb.GetUserRequest{Id: stubUser.ID})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated with invalid token, got %v", err)
	}
}

func TestGetUserWithToken(t *testing.T) {
	client := newTestClient(t)

	resp, err := client.GetUser(authContext(t), &pb.GetUserRequest{Id: stubUser.ID})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if resp.GetUser().GetEmail() != stubUser.Email {
		t.Errorf("want email %q, got %q", stubUser.Email, resp.GetUser().GetEmail())
	}
}

func TestGetUserNotFound(t *testing.T) {
	client := newTestClient(t)

	_, err := client.GetUser(authContext(t), &pb.GetUserRequest{Id: "ghost"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("want NotFound, got %v", err)
	}
}
