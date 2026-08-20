package service

import (
	"context"
	"errors"
	"strings"

	"user-management-api/internal/core/domain"
)

// In-memory fakes for the driven ports. They live in one file so each use
// case test file contains only assertions about that use case.

type mockRepo struct {
	users      map[string]*domain.User // keyed by ID
	lastCreate *domain.User
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: map[string]*domain.User{}}
}

func (m *mockRepo) Create(_ context.Context, user *domain.User) (*domain.User, error) {
	if _, err := m.GetByEmail(context.Background(), user.Email); err == nil {
		return nil, domain.ErrEmailAlreadyExists
	}
	created := *user
	created.ID = "id-" + user.Email
	m.users[created.ID] = &created
	m.lastCreate = &created
	return &created, nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockRepo) List(_ context.Context) ([]domain.User, error) {
	users := make([]domain.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, *user)
	}
	return users, nil
}

// Update mirrors the real repository: the unique index rejects an email that
// another user already holds.
func (m *mockRepo) Update(_ context.Context, id string, update domain.UserUpdate) (*domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	if update.Email != nil {
		for otherID, other := range m.users {
			if otherID != id && other.Email == *update.Email {
				return nil, domain.ErrEmailAlreadyExists
			}
		}
		user.Email = *update.Email
	}
	if update.Name != nil {
		user.Name = *update.Name
	}
	return user, nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.users[id]; !ok {
		return domain.ErrUserNotFound
	}
	delete(m.users, id)
	return nil
}

func (m *mockRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

type mockHasher struct{}

func (mockHasher) Hash(password string) (string, error) { return "hashed:" + password, nil }

func (mockHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return errors.New("mismatch")
	}
	return nil
}

type mockTokens struct{}

func (mockTokens) Generate(userID string) (string, error) { return "token-for-" + userID, nil }

func (mockTokens) Verify(token string) (string, error) {
	id, ok := strings.CutPrefix(token, "token-for-")
	if !ok {
		return "", domain.ErrInvalidToken
	}
	return id, nil
}

func newService(repo *mockRepo) *UserService {
	return NewUserService(repo, mockHasher{}, mockTokens{})
}

// registerUser is the arrange step for tests that need an existing user.
func registerUser(repo *mockRepo, email string) (*domain.User, error) {
	return newService(repo).Register(context.Background(), "Alice", email, "supersecret")
}
