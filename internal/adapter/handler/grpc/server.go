// Package grpc contains the gRPC driving adapter exposing CreateUser and
// GetUser, secured with JWT bearer tokens passed via metadata.
package grpc

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	userv1 "user-management-api/internal/adapter/handler/grpc/gen/user/v1"
	"user-management-api/internal/core/domain"
	"user-management-api/internal/core/port"
)

// Server implements userv1.UserServiceServer.
type Server struct {
	userv1.UnimplementedUserServiceServer
	service port.UserService
}

func NewServer(service port.UserService) *Server {
	return &Server{service: service}
}

func (s *Server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	user, err := s.service.Register(ctx, req.GetName(), req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.CreateUserResponse{User: toProtoUser(user)}, nil
}

func (s *Server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	user, err := s.service.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.GetUserResponse{User: toProtoUser(user)}, nil
}

func toProtoUser(user *domain.User) *userv1.User {
	return &userv1.User{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

// publicMethods lists full RPC method names that skip token validation.
// CreateUser is public so a first user can be created, mirroring the public
// HTTP register endpoint.
var publicMethods = map[string]bool{
	userv1.UserService_CreateUser_FullMethodName: true,
}

// AuthUnaryInterceptor validates the bearer token in incoming metadata for
// every non-public RPC.
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
		token := strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer "))
		if _, err := tokens.Verify(token); err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}
		return handler(ctx, req)
	}
}
