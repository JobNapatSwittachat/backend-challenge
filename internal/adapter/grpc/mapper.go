package grpc

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
	"user-management-api/internal/core/domain"
)

// toProtoUser maps a domain entity onto the generated wire type field by
// field, so nothing added to domain.User is exposed unreviewed.
func toProtoUser(user *domain.User) *userv1.User {
	return &userv1.User{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}
