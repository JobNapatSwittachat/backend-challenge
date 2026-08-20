package service

import (
	"context"
	"errors"
	"testing"

	"user-management-api/internal/core/domain"
)

func TestLoginReturnsTokenForValidCredentials(t *testing.T) {
	repo := newMockRepo()
	created, err := registerUser(repo, "a@b.com")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Email case must not matter, since registration normalized it.
	token, user, err := newService(repo).Login(context.Background(), "A@B.com", "supersecret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token != "token-for-"+created.ID {
		t.Errorf("unexpected token %q", token)
	}
	if user.ID != created.ID {
		t.Errorf("unexpected user %q", user.ID)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	repo := newMockRepo()
	if _, err := registerUser(repo, "a@b.com"); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, _, err := newService(repo).Login(context.Background(), "a@b.com", "wrongpassword")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

// An unknown email must be indistinguishable from a wrong password, or the
// endpoint becomes an account-enumeration oracle.
func TestLoginDoesNotRevealWhetherEmailExists(t *testing.T) {
	_, _, err := newService(newMockRepo()).Login(context.Background(), "ghost@b.com", "supersecret")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials (not ErrUserNotFound), got %v", err)
	}
}
