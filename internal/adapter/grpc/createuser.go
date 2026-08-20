package grpc

import (
	"context"

	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
)

// CreateUser registers a user. It is the one public RPC, mirroring the public
// HTTP register endpoint (see publicMethods in interceptor.go).
func (s *Server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	user, err := s.service.Register(ctx, req.GetName(), req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.CreateUserResponse{User: toProtoUser(user)}, nil
}
