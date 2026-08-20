package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	grpchandler "user-management-api/internal/adapter/grpc"
	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
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

// authContext returns a context carrying the given authorization metadata.
func authContext(t *testing.T, header string) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return metadata.AppendToOutgoingContext(ctx, "authorization", header)
}
