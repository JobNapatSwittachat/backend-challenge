package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"user-management-api/internal/core/domain"
)

func TestRegisterNormalizesAndHashes(t *testing.T) {
	repo := newMockRepo()

	user, err := newService(repo).Register(context.Background(), "  Alice  ", " Alice@Example.COM ", "supersecret")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if user.Name != "Alice" {
		t.Errorf("name not trimmed: got %q", user.Name)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email not normalized: got %q", user.Email)
	}
	if repo.lastCreate.PasswordHash != "hashed:supersecret" {
		t.Errorf("password was not hashed before persistence: got %q", repo.lastCreate.PasswordHash)
	}
}

func TestRegisterValidation(t *testing.T) {
	tests := []struct {
		name                  string
		userName, email, pass string
	}{
		{"empty name", "", "a@b.com", "supersecret"},
		{"name too long", strings.Repeat("x", 101), "a@b.com", "supersecret"},
		{"empty email", "Alice", "", "supersecret"},
		{"bad email", "Alice", "not-an-email", "supersecret"},
		{"email with display name", "Alice", "Alice <a@b.com>", "supersecret"},
		{"empty password", "Alice", "a@b.com", ""},
		{"short password", "Alice", "a@b.com", "1234567"},
		// 6 characters but 18 bytes: the rule counts characters, not bytes.
		{"short multibyte password", "Alice", "a@b.com", "ไทยไทย"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newService(newMockRepo()).Register(context.Background(), tc.userName, tc.email, tc.pass)
			if !errors.Is(err, domain.ErrValidation) {
				t.Errorf("want ErrValidation, got %v", err)
			}
		})
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	repo := newMockRepo()
	if _, err := registerUser(repo, "a@b.com"); err != nil {
		t.Fatalf("first register: %v", err)
	}

	_, err := newService(repo).Register(context.Background(), "Bob", "a@b.com", "supersecret")
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("want ErrEmailAlreadyExists, got %v", err)
	}
}
