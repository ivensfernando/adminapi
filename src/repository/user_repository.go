package repository

import (
	"context"
	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strategyexecutor/src/database"
	"strategyexecutor/src/model"
)

type GormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository() *GormUserRepository {
	logger.WithField("component", "GormUserExchangeRepository").
		Info("Creating new NewUserExchangeRepository with ReadOnlyDB")

	return &GormUserRepository{
		db: database.MainDB,
	}

}

func (r *GormUserRepository) GetUserByUserName(
	ctx context.Context,
	userName string,
) (*model.User, error) {

	var u model.User
	err := r.db.WithContext(ctx).
		Where("user_name = ? ", userName).
		First(&u).Error

	if err != nil {
		return nil, err
	}

	return &u, nil
}
