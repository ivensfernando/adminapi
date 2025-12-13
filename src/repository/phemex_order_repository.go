package repository

import (
	"context"
	"errors"

	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"adminapi/src/database"
	"adminapi/src/model"
)

// PhemexOrderRepository handles persistence for PhemexOrder entities.
type PhemexOrderRepository struct {
	db *gorm.DB
}

// NewPhemexOrderRepository creates a new repository instance using the main read/write database.
func NewPhemexOrderRepository() *PhemexOrderRepository {
	logger.WithField("component", "PhemexOrderRepository").
		Info("Creating new PhemexOrderRepository with MainDB")

	return &PhemexOrderRepository{
		db: database.MainDB,
	}
}

// WithDB allows overriding the underlying *gorm.DB instance.
// Useful for tests or when using a specific session/transaction.
func (r *PhemexOrderRepository) WithDB(db *gorm.DB) *PhemexOrderRepository {
	logger.WithField("component", "PhemexOrderRepository").
		Debug("Creating PhemexOrderRepository with custom DB instance")

	return &PhemexOrderRepository{db: db}
}

// ---------------------------------------------------
// Basic CRUD methods
// ---------------------------------------------------

// Create inserts a new PhemexOrder document into the database.
// The given entity will be updated with the generated ID and timestamps.
func (r *PhemexOrderRepository) Create(
	ctx context.Context,
	order *model.PhemexOrder,
) error {

	logger.WithFields(map[string]interface{}{
		"repo":   "PhemexOrderRepository",
		"op":     "Create",
		"symbol": order.Symbol,
		"side":   order.Side,
		"qty":    order.OrderQty,
	}).Debug("Creating new Phemex order")

	err := r.db.WithContext(ctx).Create(order).Error
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":   "PhemexOrderRepository",
			"op":     "Create",
			"symbol": order.Symbol,
			"side":   order.Side,
		}).WithError(err).Error("Failed to create Phemex order")

		return err
	}

	logger.WithFields(map[string]interface{}{
		"repo":     "PhemexOrderRepository",
		"op":       "Create",
		"order_id": order.ID,
	}).Info("Phemex order created successfully")

	return nil
}

// FindByID fetches a single PhemexOrder by its primary ID.
// Returns (nil, nil) if not found.
func (r *PhemexOrderRepository) FindByID(
	ctx context.Context,
	id uint,
) (*model.PhemexOrder, error) {

	logger.WithFields(map[string]interface{}{
		"repo": "PhemexOrderRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("Fetching Phemex order by ID")

	var order model.PhemexOrder

	err := r.db.WithContext(ctx).
		First(&order, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo": "PhemexOrderRepository",
				"op":   "FindByID",
				"id":   id,
			}).Info("Phemex order not found")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo": "PhemexOrderRepository",
			"op":   "FindByID",
			"id":   id,
		}).WithError(err).Error("Failed to fetch Phemex order by ID")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo": "PhemexOrderRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("Phemex order fetched successfully")

	return &order, nil
}

// FindByOrderID fetches a PhemexOrder by its external OrderID (Phemex order ID).
// Returns (nil, nil) if not found.
func (r *PhemexOrderRepository) FindByOrderID(
	ctx context.Context,
	orderID string,
) (*model.PhemexOrder, error) {

	logger.WithFields(map[string]interface{}{
		"repo":        "PhemexOrderRepository",
		"op":          "FindByOrderID",
		"external_id": orderID,
	}).Debug("Fetching Phemex order by external order ID")

	var order model.PhemexOrder

	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo":        "PhemexOrderRepository",
				"op":          "FindByOrderID",
				"external_id": orderID,
			}).Info("Phemex order not found by external order ID")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo":        "PhemexOrderRepository",
			"op":          "FindByOrderID",
			"external_id": orderID,
		}).WithError(err).Error("Failed to fetch Phemex order by external order ID")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "PhemexOrderRepository",
		"op":          "FindByOrderID",
		"external_id": orderID,
	}).Debug("Phemex order fetched by external order ID successfully")

	return &order, nil
}

// ---------------------------------------------------
// Query helpers
// ---------------------------------------------------

// FindLatest returns the latest Phemex orders ordered from newest to oldest.
func (r *PhemexOrderRepository) FindLatest(
	ctx context.Context,
	limit int,
) ([]model.PhemexOrder, error) {

	if limit <= 0 {
		limit = 20
	}

	logger.WithFields(map[string]interface{}{
		"repo":  "PhemexOrderRepository",
		"op":    "FindLatest",
		"limit": limit,
	}).Debug("Fetching latest Phemex orders")

	var orders []model.PhemexOrder

	err := r.db.WithContext(ctx).
		Order("id DESC").
		Limit(limit).
		Find(&orders).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":  "PhemexOrderRepository",
			"op":    "FindLatest",
			"limit": limit,
		}).WithError(err).Error("Failed to fetch latest Phemex orders")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "PhemexOrderRepository",
		"op":          "FindLatest",
		"limit":       limit,
		"rows_return": len(orders),
	}).Info("Latest Phemex orders fetched")

	return orders, nil
}

// FindLatestBySymbol returns the latest Phemex orders for a given symbol,
// ordered from newest to oldest.
func (r *PhemexOrderRepository) FindLatestBySymbol(
	ctx context.Context,
	symbol string,
	limit int,
) ([]model.PhemexOrder, error) {

	if limit <= 0 {
		limit = 20
	}

	logger.WithFields(map[string]interface{}{
		"repo":   "PhemexOrderRepository",
		"op":     "FindLatestBySymbol",
		"symbol": symbol,
		"limit":  limit,
	}).Debug("Fetching latest Phemex orders by symbol")

	var orders []model.PhemexOrder

	err := r.db.WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("id DESC").
		Limit(limit).
		Find(&orders).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":   "PhemexOrderRepository",
			"op":     "FindLatestBySymbol",
			"symbol": symbol,
			"limit":  limit,
		}).WithError(err).Error("Failed to fetch latest Phemex orders by symbol")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "PhemexOrderRepository",
		"op":          "FindLatestBySymbol",
		"symbol":      symbol,
		"limit":       limit,
		"rows_return": len(orders),
	}).Info("Latest Phemex orders by symbol fetched")

	return orders, nil
}
