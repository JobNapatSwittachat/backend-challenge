// Package grpc contains the gRPC driving adapter. Each RPC lives in its own
// file, mirroring the HTTP adapter's layout; authentication is in
// interceptor.go and the shared mapping helpers in mapper.go.
package grpc

import (
	userv1 "user-management-api/internal/adapter/grpc/gen/user/v1"
	"user-management-api/internal/core/port"
)

// Server implements the generated userv1.UserServiceServer.
type Server struct {
	userv1.UnimplementedUserServiceServer
	service port.UserService
}

func NewServer(service port.UserService) *Server {
	return &Server{service: service}
}
