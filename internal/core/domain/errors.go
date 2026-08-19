package domain

import "errors"

// Sentinel errors returned by the core layer. Adapters translate driver
// specific errors into these so the rest of the app never sees driver types.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrValidation         = errors.New("validation failed")
)
