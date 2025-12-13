package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	handler2 "vsC1Y2025V01/src/handler"
	"vsC1Y2025V01/src/repository"

	"vsC1Y2025V01/src/model"

	"github.com/sirupsen/logrus"
)

type inMemoryUserRepository struct {
	mu     sync.Mutex
	nextID uint
	users  map[uint]*model.User
}

func newInMemoryUserRepository() *inMemoryUserRepository {
	return &inMemoryUserRepository{users: make(map[uint]*model.User)}
}

func (r *inMemoryUserRepository) Create(user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	now := time.Now()
	clone := *user
	clone.ID = r.nextID
	clone.CreatedAt = now
	clone.UpdatedAt = now
	r.users[clone.ID] = &clone

	user.ID = clone.ID
	user.CreatedAt = clone.CreatedAt
	user.UpdatedAt = clone.UpdatedAt
	return nil
}

func (r *inMemoryUserRepository) FindByUsername(username string) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, user := range r.users {
		if user.Username == username {
			clone := *user
			return &clone, nil
		}
	}

	return nil, repository.ErrUserNotFound
}

func (r *inMemoryUserRepository) FindByID(id uint) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}

	clone := *user
	return &clone, nil
}

func (r *inMemoryUserRepository) Update(user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.users[user.ID]
	if !ok {
		return repository.ErrUserNotFound
	}

	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now()
	}

	stored.Username = user.Username
	stored.Password = user.Password
	stored.CreatedAt = user.CreatedAt
	stored.UpdatedAt = user.UpdatedAt
	stored.LastLogin = user.LastLogin
	stored.LastSeen = user.LastSeen
	return nil
}

func TestRegisterAndLogin(t *testing.T) {
	logger := logrus.NewEntry(logrus.StandardLogger())

	repo := newInMemoryUserRepository()
	repository.SetUserRepository(repo)
	t.Cleanup(func() {
		repository.SetUserRepository(nil)
	})

	// Setup test user payload
	payload := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	body, _ := json.Marshal(payload)

	// Test registration
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler := handler2.RegisterHandler(logger)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	// Test login
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	handler2.LoginHandler(logger).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
