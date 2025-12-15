package repository

import (
	"errors"
	"sync"

	"adminapi/src/database"
	"adminapi/src/model"

	"gorm.io/gorm"
)

var (
	ErrExchangeNotFound     = errors.New("exchange not found")
	ErrUserExchangeNotFound = errors.New("user exchange not found")
)

type UserExchangeStore interface {
	CreateExchange(exchange *model.Exchange) error
	GetExchangeByID(id uint) (*model.Exchange, error)
	FindUserExchange(userID, exchangeID uint) (*model.UserExchange, error)
	SaveUserExchange(ue *model.UserExchange) error
	ListFormUserExchanges(userID uint) ([]model.UserExchange, error)
	DeleteUserExchange(userID, exchangeID uint) (bool, error)
}

var (
	storeMu sync.RWMutex
	store   UserExchangeStore = &gormUserExchangeRepository{}
)

func SetUserExchangeStore(s UserExchangeStore) {
	storeMu.Lock()
	defer storeMu.Unlock()

	if s == nil {
		store = &gormUserExchangeRepository{}
		return
	}

	store = s
}

func GetUserExchangeStore() UserExchangeStore {
	storeMu.RLock()
	current := store
	storeMu.RUnlock()

	if current != nil {
		return current
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	if store == nil {
		store = &gormUserExchangeRepository{}
	}

	return store
}

type gormUserExchangeRepository struct{}

func (s *gormUserExchangeRepository) CreateExchange(exchange *model.Exchange) error {
	if database.MainDB == nil {
		return errors.New("database connection is not initialized")
	}

	return database.MainDB.Create(exchange).Error
}

func (s *gormUserExchangeRepository) GetExchangeByID(id uint) (*model.Exchange, error) {
	if database.MainDB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	var exchange model.Exchange
	if err := database.MainDB.First(&exchange, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExchangeNotFound
		}

		return nil, err
	}

	return &exchange, nil
}

func (s *gormUserExchangeRepository) FindUserExchange(userID, exchangeID uint) (*model.UserExchange, error) {
	if database.MainDB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	var userExchange model.UserExchange
	if err := database.MainDB.Where("user_id = ? AND exchange_id = ?", userID, exchangeID).Preload("Exchange").First(&userExchange).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserExchangeNotFound
		}

		return nil, err
	}

	return &userExchange, nil
}

func (s *gormUserExchangeRepository) SaveUserExchange(ue *model.UserExchange) error {
	if database.MainDB == nil {
		return errors.New("database connection is not initialized")
	}

	return database.MainDB.Save(ue).Error
}

func (s *gormUserExchangeRepository) ListFormUserExchanges(userID uint) ([]model.UserExchange, error) {
	if database.MainDB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	var exchanges []model.UserExchange
	if err := database.MainDB.Preload("Exchange").Where("user_id = ? AND show_in_forms = ?", userID, true).Find(&exchanges).Error; err != nil {
		return nil, err
	}

	return exchanges, nil
}

func (s *gormUserExchangeRepository) DeleteUserExchange(userID, exchangeID uint) (bool, error) {
	if database.MainDB == nil {
		return false, errors.New("database connection is not initialized")
	}

	res := database.MainDB.Where("user_id = ? AND exchange_id = ?", userID, exchangeID).Delete(&model.UserExchange{})
	if res.Error != nil {
		return false, res.Error
	}

	return res.RowsAffected > 0, nil
}
