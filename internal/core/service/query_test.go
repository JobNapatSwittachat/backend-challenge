package service

import (
	"context"
	"errors"
	"testing"

	"user-management-api/internal/core/domain"
)

func TestGetByIDAndDelete(t *testing.T) {
	repo := newMockRepo()
	svc := newService(repo)
	ctx := context.Background()

	created, err := registerUser(repo, "a@b.com")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := svc.GetByID(ctx, created.ID); err != nil {
		t.Fatalf("get: %v", err)
	}
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.GetByID(ctx, created.ID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("want ErrUserNotFound after delete, got %v", err)
	}
	if err := svc.Delete(ctx, created.ID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("want ErrUserNotFound on double delete, got %v", err)
	}
}

func TestListAndCount(t *testing.T) {
	repo := newMockRepo()
	svc := newService(repo)
	ctx := context.Background()

	for _, email := range []string{"a@b.com", "b@b.com", "c@b.com"} {
		if _, err := registerUser(repo, email); err != nil {
			t.Fatalf("register %s: %v", email, err)
		}
	}

	users, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("want 3 users, got %d", len(users))
	}

	count, err := svc.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("want count 3, got %d", count)
	}
}
