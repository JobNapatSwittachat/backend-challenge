package service

import (
	"context"
	"errors"
	"testing"

	"user-management-api/internal/core/domain"
)

func TestUpdateNormalizesFields(t *testing.T) {
	repo := newMockRepo()
	created, err := registerUser(repo, "a@b.com")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	newName := "  Alice Smith "
	newEmail := "NEW@b.com"
	updated, err := newService(repo).Update(context.Background(), created.ID, domain.UserUpdate{
		Name:  &newName,
		Email: &newEmail,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Alice Smith" {
		t.Errorf("name not trimmed: %q", updated.Name)
	}
	if updated.Email != "new@b.com" {
		t.Errorf("email not normalized: %q", updated.Email)
	}
}

func TestUpdateRequiresAtLeastOneField(t *testing.T) {
	_, err := newService(newMockRepo()).Update(context.Background(), "any", domain.UserUpdate{})
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("want ErrValidation, got %v", err)
	}
}

func TestUpdateRejectsInvalidEmail(t *testing.T) {
	bad := "nope"
	_, err := newService(newMockRepo()).Update(context.Background(), "any", domain.UserUpdate{Email: &bad})
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("want ErrValidation, got %v", err)
	}
}

func TestUpdateRejectsEmailTakenByAnotherUser(t *testing.T) {
	repo := newMockRepo()
	first, err := registerUser(repo, "first@b.com")
	if err != nil {
		t.Fatalf("register first: %v", err)
	}
	if _, err := registerUser(repo, "second@b.com"); err != nil {
		t.Fatalf("register second: %v", err)
	}

	taken := "second@b.com"
	_, err = newService(repo).Update(context.Background(), first.ID, domain.UserUpdate{Email: &taken})
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("want ErrEmailAlreadyExists, got %v", err)
	}
}

func TestUpdateUnknownUser(t *testing.T) {
	name := "Ghost"
	_, err := newService(newMockRepo()).Update(context.Background(), "missing", domain.UserUpdate{Name: &name})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}
