package service

import (
	"context"

	"user-management-api/internal/core/domain"
)

// This file holds the read-only use cases, which are thin pass-throughs to
// the repository: they carry no rules of their own, and giving each its own
// file would spread three one-line methods over three files.

// GetByID fetches a single user.
func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns every user.
func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}

// Count reports how many users exist; used by the background counter.
func (s *UserService) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
