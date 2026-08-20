package service

import "context"

// Delete removes a user, returning domain.ErrUserNotFound if the id matches
// no existing user.
func (s *UserService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
