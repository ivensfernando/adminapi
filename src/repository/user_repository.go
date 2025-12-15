package repository

import (
	"errors"
	"sync"
	"time"

	"adminapi/src/database"
	"adminapi/src/model"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists")

type UserRepository interface {
	Create(user *model.User) error
	FindByUsername(username string) (*model.User, error)
	FindByID(id uint) (*model.User, error)
	Update(user *model.User) error
}

var (
	userRepo   UserRepository = &gormUserRepository{}
	userRepoMu sync.RWMutex
)

func SetUserRepository(repo UserRepository) {
	userRepoMu.Lock()
	defer userRepoMu.Unlock()

	if repo == nil {
		userRepo = &gormUserRepository{}
		return
	}

	userRepo = repo
}

func GetUserRepository() UserRepository {
	userRepoMu.RLock()
	repo := userRepo
	userRepoMu.RUnlock()

	if repo != nil {
		return repo
	}

	userRepoMu.Lock()
	defer userRepoMu.Unlock()

	if userRepo == nil {
		userRepo = &gormUserRepository{}
	}

	return userRepo
}

type gormUserRepository struct{}

func (r *gormUserRepository) Create(user *model.User) error {
	if database.MainDB == nil {
		return errors.New("database connection is not initialized")
	}
	if err := database.MainDB.Create(user).Error; err != nil {
		var errWithSQLState interface{ SQLState() string }

		switch {
		case errors.Is(err, gorm.ErrDuplicatedKey):
			return ErrUserAlreadyExists
		case errors.As(err, &errWithSQLState) && errWithSQLState.SQLState() == "23505":
			return ErrUserAlreadyExists
		default:
			return err
		}
	}

	return nil
}

func (r *gormUserRepository) FindByUsername(username string) (*model.User, error) {
	if database.MainDB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	var user model.User
	if err := database.MainDB.Where("user_name = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (r *gormUserRepository) FindByID(id uint) (*model.User, error) {
	if database.MainDB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	var user model.User
	if err := database.MainDB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (r *gormUserRepository) Update(user *model.User) error {
	if database.MainDB == nil {
		return errors.New("database connection is not initialized")
	}

	if user != nil {
		user.UpdatedAt = time.Now()
	}

	return database.MainDB.Save(user).Error
}
