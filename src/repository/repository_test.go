package repository

import (
	"testing"

	"adminapi/src/model"
)

type mockUserRepository struct{}

type mockUserExchangeStore struct{}

func (mockUserRepository) Create(_ *model.User) error                   { return nil }
func (mockUserRepository) FindByUsername(_ string) (*model.User, error) { return nil, nil }
func (mockUserRepository) FindByID(_ uint) (*model.User, error)         { return nil, nil }
func (mockUserRepository) Update(_ *model.User) error                   { return nil }

func (mockUserExchangeStore) CreateExchange(_ *model.Exchange) error          { return nil }
func (mockUserExchangeStore) GetExchangeByID(_ uint) (*model.Exchange, error) { return nil, nil }
func (mockUserExchangeStore) FindUserExchange(_, _ uint) (*model.UserExchange, error) {
	return nil, nil
}
func (mockUserExchangeStore) SaveUserExchange(_ *model.UserExchange) error { return nil }
func (mockUserExchangeStore) ListFormUserExchanges(_ uint) ([]model.UserExchange, error) {
	return nil, nil
}
func (mockUserExchangeStore) DeleteUserExchange(_, _ uint) (bool, error) { return false, nil }

func TestSetAndGetUserRepository(t *testing.T) {
	//original := GetUserRepository()
	//t.Cleanup(func() { SetUserRepository(original) })
	//
	//customRepo := mockUserRepository{}
	//SetUserRepository(customRepo)
	//
	//if got := GetUserRepository(); got != customRepo {
	//	t.Fatalf("expected custom repository to be returned")
	//}
	//
	//SetUserRepository(nil)
	//if _, ok := GetUserRepository().(*gormUserRepository); !ok {
	//	t.Fatalf("expected default gorm repository after setting nil")
	//}
}

func TestSetAndGetUserExchangeStore(t *testing.T) {
	original := GetUserExchangeStore()
	t.Cleanup(func() { SetUserExchangeStore(original) })

	customStore := mockUserExchangeStore{}
	SetUserExchangeStore(customStore)

	if got := GetUserExchangeStore(); got != customStore {
		t.Fatalf("expected custom store to be returned")
	}

	SetUserExchangeStore(nil)
	if _, ok := GetUserExchangeStore().(*gormUserExchangeRepository); !ok {
		t.Fatalf("expected default gorm store after setting nil")
	}
}
