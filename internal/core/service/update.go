package service

import (
	"context"
	"fmt"
	"strings"

	"user-management-api/internal/core/domain"
)

// Update changes a user's name and/or email. A nil field is left unchanged,
// and the same validation rules as registration apply to the fields present.
func (s *UserService) Update(ctx context.Context, id string, update domain.UserUpdate) (*domain.User, error) {
	if update.Name == nil && update.Email == nil {
		return nil, fmt.Errorf("%w: at least one of name or email is required", domain.ErrValidation)
	}

	if update.Name != nil {
		trimmed := strings.TrimSpace(*update.Name)
		if err := validateName(trimmed); err != nil {
			return nil, err
		}
		update.Name = &trimmed
	}

	if update.Email != nil {
		normalized := normalizeEmail(*update.Email)
		if err := validateEmail(normalized); err != nil {
			return nil, err
		}
		update.Email = &normalized
	}

	return s.repo.Update(ctx, id, update)
}
