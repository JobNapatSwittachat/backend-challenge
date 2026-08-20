package grpc

import (
	"context"

	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
)

// GetUser fetches a user by ID. It requires a bearer token in metadata.
func (s *Server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	user, err := s.service.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userv1.GetUserResponse{User: toProtoUser(user)}, nil
}
