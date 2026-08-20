package service

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"user-management-api/internal/core/domain"
)

// Input rules shared by every use case, so HTTP and gRPC enforce them
// identically.
const (
	minPasswordLength = 8
	maxNameLength     = 100
)

// normalizeEmail makes lookups and uniqueness case-insensitive.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return fmt.Errorf("%w: name must be at most %d characters", domain.ErrValidation, maxNameLength)
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("%w: email is required", domain.ErrValidation)
	}
	// mail.ParseAddress also accepts "Name <a@b.com>", so require the parsed
	// address to be exactly what was sent.
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return fmt.Errorf("%w: email format is invalid", domain.ErrValidation)
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("%w: password is required", domain.ErrValidation)
	}
	if utf8.RuneCountInString(password) < minPasswordLength {
		return fmt.Errorf("%w: password must be at least %d characters", domain.ErrValidation, minPasswordLength)
	}
	return nil
}
