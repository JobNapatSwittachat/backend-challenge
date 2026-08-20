package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
	"user-management-api/internal/adapter/http/middleware"
	"user-management-api/internal/core/port"
)

// publicMethods lists the RPCs that skip token validation. CreateUser is
// public so a first user can be created, mirroring HTTP register.
var publicMethods = map[string]bool{
	userv1.UserService_CreateUser_FullMethodName: true,
}

// AuthUnaryInterceptor validates the bearer token in incoming metadata for
// every non-public RPC. It is the gRPC counterpart of middleware.Auth.
func AuthUnaryInterceptor(tokens port.TokenService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
		}

		// Shared with the HTTP middleware so both transports accept exactly
		// the same credential format.
		token, ok := middleware.BearerToken(values[0])
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "malformed authorization metadata")
		}
		if _, err := tokens.Verify(token); err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}
		return handler(ctx, req)
	}
}
