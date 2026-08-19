// Package port defines the interfaces (ports) between the core business
// logic and the outside world. Driven adapters (MongoDB, bcrypt, JWT)
// implement the driven ports; driving adapters (HTTP, gRPC) consume the
// service port.
package port

import (
	"context"

	"user-management-api/internal/core/domain"
)

// UserRepository is the driven port for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id string, update domain.UserUpdate) (*domain.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// PasswordHasher is the driven port for password hashing.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TokenService is the driven port for issuing and verifying auth tokens.
type TokenService interface {
	Generate(userID string) (string, error)
	// Verify returns the subject (user ID) encoded in the token.
	Verify(token string) (string, error)
}

// UserService is the driving port exposing the application's use cases.
type UserService interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (token string, user *domain.User, err error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id string, update domain.UserUpdate) (*domain.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}
