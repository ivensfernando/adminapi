package repository

import (
	"adminapi/src/database"
	"context"
	"errors"

	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"adminapi/src/model"
)

// gormExchangeRepository implements exchange persistence using GORM.
type GormExchangeRepository struct {
	db *gorm.DB
}

// NewExchangeRepository creates a new Exchange repository using the given gorm DB.
func NewExchangeRepository() *GormExchangeRepository {
	logger.WithField("component", "ExchangeRepository").
		Info("Creating new ExchangeRepository with custom DB instance")

	return &GormExchangeRepository{
		db: database.MainDB,
	}
}

// CreateExchange inserts a new exchange into the database.
func (s *GormExchangeRepository) CreateExchange(
	ctx context.Context,
	exchange *model.Exchange,
) error {

	logger.WithFields(map[string]interface{}{
		"repo": "ExchangeRepository",
		"op":   "CreateExchange",
		"name": exchange.Name,
	}).Debug("Creating new exchange")

	err := s.db.WithContext(ctx).Create(exchange).Error
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo": "ExchangeRepository",
			"op":   "CreateExchange",
			"name": exchange.Name,
		}).WithError(err).Error("Failed to create exchange")

		return err
	}

	logger.WithFields(map[string]interface{}{
		"repo":       "ExchangeRepository",
		"op":         "CreateExchange",
		"exchangeID": exchange.ID,
		"name":       exchange.Name,
	}).Info("Exchange created successfully")

	return nil
}

// FindByID fetches an exchange by its primary ID.
// Returns (nil, nil) if not found.
func (s *GormExchangeRepository) FindByID(
	ctx context.Context,
	id uint,
) (*model.Exchange, error) {

	logger.WithFields(map[string]interface{}{
		"repo": "ExchangeRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("Fetching exchange by ID")

	var exchange model.Exchange

	err := s.db.WithContext(ctx).
		First(&exchange, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo": "ExchangeRepository",
				"op":   "FindByID",
				"id":   id,
			}).Info("Exchange not found by ID")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo": "ExchangeRepository",
			"op":   "FindByID",
			"id":   id,
		}).WithError(err).Error("Failed to fetch exchange by ID")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo": "ExchangeRepository",
		"op":   "FindByID",
		"id":   id,
		"name": exchange.Name,
	}).Debug("Exchange fetched successfully by ID")

	return &exchange, nil
}

// FindByName fetches an exchange by its name.
// Returns (nil, nil) if not found.
func (s *GormExchangeRepository) FindByName(
	ctx context.Context,
	name string,
) (*model.Exchange, error) {

	logger.WithFields(map[string]interface{}{
		"repo": "ExchangeRepository",
		"op":   "FindByName",
		"name": name,
	}).Debug("Fetching exchange by name")

	var exchange model.Exchange

	err := s.db.WithContext(ctx).
		Where("name = ?", name).
		First(&exchange).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo": "ExchangeRepository",
				"op":   "FindByName",
				"name": name,
			}).Info("Exchange not found by name")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo": "ExchangeRepository",
			"op":   "FindByName",
			"name": name,
		}).WithError(err).Error("Failed to fetch exchange by name")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo": "ExchangeRepository",
		"op":   "FindByName",
		"id":   exchange.ID,
		"name": exchange.Name,
	}).Debug("Exchange fetched successfully by name")

	return &exchange, nil
}
