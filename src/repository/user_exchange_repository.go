package repository

import (
	"context"
	"strategyexecutor/src/database"
	"strategyexecutor/src/model"

	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

type UserExchangeRepository interface {
	Create(ctx context.Context, ue *model.UserExchange) error
	GetByUserAndExchange(ctx context.Context, userID string, exchangeID uint) (*model.UserExchange, error)
	Update(ctx context.Context, ue *model.UserExchange) error
	UpdateByUserAndExchange(ctx context.Context, userID string, exchangeID uint, updates map[string]interface{}) error
}

type GormUserExchangeRepository struct {
	db *gorm.DB
}

func NewUserExchangeRepository() *GormUserExchangeRepository {
	logger.WithField("component", "GormUserExchangeRepository").
		Info("Creating new NewUserExchangeRepository with ReadOnlyDB")

	return &GormUserExchangeRepository{
		db: database.MainDB,
	}
}

// Create inserts a new UserExchange record.
func (r *GormUserExchangeRepository) Create(ctx context.Context, ue *model.UserExchange) error {
	return r.db.WithContext(ctx).Create(ue).Error
}

// GetByUserAndExchange returns a UserExchange for the given userID and exchangeID.
func (r *GormUserExchangeRepository) GetByUserAndExchange(
	ctx context.Context,
	userID uint,
	exchangeID uint,
) (*model.UserExchange, error) {

	var ue model.UserExchange
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND exchange_id = ?", userID, exchangeID).
		First(&ue).Error

	if err != nil {
		return nil, err
	}

	return &ue, nil
}

// MarkNoTradeWindowOrdersClosed sets no_trade_window_orders_closed = true
// for the given userID + exchangeID.
func (r *GormUserExchangeRepository) MarkNoTradeWindowOrdersClosed(
	ctx context.Context,
	userID uint,
	exchangeID uint,
) error {
	res := r.db.WithContext(ctx).
		Model(&model.UserExchange{}).
		Where("user_id = ? AND exchange_id = ?", userID, exchangeID).
		Update("no_trade_window_orders_closed", true)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// GetUserRunOnServerAndPercent returns only the fields needed for runtime checks.

// Update updates an existing UserExchange using its primary key (ID).
func (r *GormUserExchangeRepository) Update(ctx context.Context, ue *model.UserExchange) error {
	// Save will update all fields, including zero values.
	return r.db.WithContext(ctx).Save(ue).Error
}

// UpdateByUserAndExchange updates an existing UserExchange using the composite key (user_id, exchange_id).
// The updates parameter is a map of columns to new values.
func (r *GormUserExchangeRepository) UpdateByUserAndExchange(
	ctx context.Context,
	userID string,
	exchangeID uint,
	updates map[string]interface{},
) error {

	return r.db.WithContext(ctx).
		Model(&model.UserExchange{}).
		Where("user_id = ? AND exchange_id = ?", userID, exchangeID).
		Updates(updates).Error
}

// Upsert creates a new UserExchange or updates API keys if the (user_id, exchange_id)
// combination already exists.
func (r *GormUserExchangeRepository) Upsert(
	ctx context.Context,
	ue *model.UserExchange,
) error {

	// OnConflict: match on composite unique index (user_id, exchange_id)
	// If a record already exists, update the credential fields and ShowInForms.
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "exchange_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"api_key",
				"api_secret",
				"api_passphrase",
				"updated_at",
			}),
		}).
		Create(ue).Error
}
