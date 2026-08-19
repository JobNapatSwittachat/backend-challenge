package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"user-management-api/internal/core/domain"
)

// --- Mocks (hand-written, standard library only) ---

type mockRepo struct {
	users      map[string]*domain.User // keyed by ID
	createErr  error
	lastCreate *domain.User
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: map[string]*domain.User{}}
}

func (m *mockRepo) Create(_ context.Context, user *domain.User) (*domain.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	for _, existing := range m.users {
		if existing.Email == user.Email {
			return nil, domain.ErrEmailAlreadyExists
		}
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

func (m *mockRepo) Update(_ context.Context, id string, update domain.UserUpdate) (*domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	if update.Name != nil {
		user.Name = *update.Name
	}
	if update.Email != nil {
		user.Email = *update.Email
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

type mockTokens struct {
	generateErr error
}

func (m mockTokens) Generate(userID string) (string, error) {
	if m.generateErr != nil {
		return "", m.generateErr
	}
	return "token-for-" + userID, nil
}

func (m mockTokens) Verify(token string) (string, error) {
	id, ok := strings.CutPrefix(token, "token-for-")
	if !ok {
		return "", domain.ErrInvalidToken
	}
	return id, nil
}

func newService(repo *mockRepo) *UserService {
	return NewUserService(repo, mockHasher{}, mockTokens{})
}

// --- Register ---

func TestRegisterSuccess(t *testing.T) {
	repo := newMockRepo()
	svc := newService(repo)

	user, err := svc.Register(context.Background(), "  Alice  ", " Alice@Example.COM ", "supersecret")
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

func TestRegisterDuplicateEmail(t *testing.T) {
	repo := newMockRepo()
	svc := newService(repo)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "Alice", "a@b.com", "supersecret"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := svc.Register(ctx, "Bob", "a@b.com", "supersecret")
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("want ErrEmailAlreadyExists, got %v", err)
	}
}

// --- Login ---

func TestLoginSuccess(t *testing.T) {
	repo := newMockRepo()
	svc := newService(repo)
	ctx := context.Background()

	created, err := svc.Register(ctx, "Alice", "a@b.com", "supersecret")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	token, user, err := svc.Login(ctx, "A@B.com", "supersecret")
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

func TestLoginWrongPassword(t *testing.T) {
	svc := newService(newMockRepo())
	ctx := context.Background()
	if _, err := svc.Register(ctx, "Alice", "a@b.com", "supersecret"); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, _, err := svc.Login(ctx, "a@b.com", "wrongpassword")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnknownEmailDoesNotLeakExistence(t *testing.T) {
	_, _, err := newService(newMockRepo()).Login(context.Background(), "ghost@b.com", "supersecret")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials (not ErrUserNotFound), got %v", err)
	}
}

// --- Update ---

func TestUpdateNameAndEmail(t *testing.T) {
	repo := newMockRepo()
	svc := newService(repo)
	ctx := context.Background()

	created, err := svc.Register(ctx, "Alice", "a@b.com", "supersecret")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	newName := "  Alice Smith "
	newEmail := "NEW@b.com"
	updated, err := svc.Update(ctx, created.ID, domain.UserUpdate{Name: &newName, Email: &newEmail})
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

func TestUpdateInvalidEmail(t *testing.T) {
	bad := "nope"
	_, err := newService(newMockRepo()).Update(context.Background(), "any", domain.UserUpdate{Email: &bad})
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("want ErrValidation, got %v", err)
	}
}

// --- Delete / Get / List / Count ---

func TestDeleteAndGet(t *testing.T) {
	repo := newMockRepo()
	svc := newService(repo)
	ctx := context.Background()

	created, err := svc.Register(ctx, "Alice", "a@b.com", "supersecret")
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
		if _, err := svc.Register(ctx, "User", email, "supersecret"); err != nil {
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
