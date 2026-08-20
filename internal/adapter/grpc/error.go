package grpc

import (
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"user-management-api/internal/core/domain"
)

// toGRPCError maps the core layer's sentinel errors onto gRPC status codes —
// the gRPC counterpart of response.DomainError. Unrecognized errors are
// logged and reported as Internal so internal details never reach clients.
func toGRPCError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		slog.Error("internal error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
