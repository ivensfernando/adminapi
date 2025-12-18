package repository

import (
	"context"
	"errors"

	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"strategyexecutor/src/database"
	"strategyexecutor/src/model"
)

// KucoinOrderRepository handles persistence for KucoinOrder entities.
type KucoinOrderRepository struct {
	db *gorm.DB
}

// NewKucoinOrderRepository creates a new repository instance using the main read/write database.
func NewKucoinOrderRepository() *KucoinOrderRepository {
	logger.WithField("component", "KucoinOrderRepository").
		Info("Creating new KucoinOrderRepository with MainDB")

	return &KucoinOrderRepository{db: database.MainDB}
}

// WithDB allows overriding the underlying *gorm.DB instance.
func (r *KucoinOrderRepository) WithDB(db *gorm.DB) *KucoinOrderRepository {
	logger.WithField("component", "KucoinOrderRepository").
		Debug("Creating KucoinOrderRepository with custom DB instance")

	return &KucoinOrderRepository{db: db}
}

// Create inserts a new KuCoin order into the database.
func (r *KucoinOrderRepository) Create(ctx context.Context, order *model.KucoinOrder) error {
	logger.WithFields(map[string]interface{}{
		"repo":   "KucoinOrderRepository",
		"op":     "Create",
		"symbol": order.Symbol,
		"side":   order.Side,
		"qty":    order.Size,
	}).Debug("Creating new KuCoin order")

	if err := r.db.WithContext(ctx).Create(order).Error; err != nil {
		logger.WithFields(map[string]interface{}{
			"repo": "KucoinOrderRepository",
			"op":   "Create",
		}).WithError(err).Error("Failed to create KuCoin order")
		return err
	}

	logger.WithFields(map[string]interface{}{
		"repo":     "KucoinOrderRepository",
		"op":       "Create",
		"order_id": order.ID,
	}).Info("KuCoin order created successfully")

	return nil
}

// FindByID fetches a single KuCoin order by its primary ID.
func (r *KucoinOrderRepository) FindByID(ctx context.Context, id uint) (*model.KucoinOrder, error) {
	logger.WithFields(map[string]interface{}{
		"repo": "KucoinOrderRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("Fetching KuCoin order by ID")

	var order model.KucoinOrder
	err := r.db.WithContext(ctx).First(&order, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo": "KucoinOrderRepository",
				"op":   "FindByID",
				"id":   id,
			}).Info("KuCoin order not found")
			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo": "KucoinOrderRepository",
			"op":   "FindByID",
			"id":   id,
		}).WithError(err).Error("Failed to fetch KuCoin order by ID")
		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo": "KucoinOrderRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("KuCoin order fetched successfully")

	return &order, nil
}

// FindByOrderID fetches a KuCoin order by its exchange OrderID.
func (r *KucoinOrderRepository) FindByOrderID(ctx context.Context, orderID string) (*model.KucoinOrder, error) {
	logger.WithFields(map[string]interface{}{
		"repo":        "KucoinOrderRepository",
		"op":          "FindByOrderID",
		"external_id": orderID,
	}).Debug("Fetching KuCoin order by external order ID")

	var order model.KucoinOrder
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo":        "KucoinOrderRepository",
				"op":          "FindByOrderID",
				"external_id": orderID,
			}).Info("KuCoin order not found by external order ID")
			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo":        "KucoinOrderRepository",
			"op":          "FindByOrderID",
			"external_id": orderID,
		}).WithError(err).Error("Failed to fetch KuCoin order by external order ID")
		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "KucoinOrderRepository",
		"op":          "FindByOrderID",
		"external_id": orderID,
	}).Debug("KuCoin order fetched by external order ID successfully")

	return &order, nil
}

// FindLatest returns the latest KuCoin orders ordered from newest to oldest.
func (r *KucoinOrderRepository) FindLatest(ctx context.Context, limit int) ([]model.KucoinOrder, error) {
	if limit <= 0 {
		limit = 20
	}

	logger.WithFields(map[string]interface{}{
		"repo":  "KucoinOrderRepository",
		"op":    "FindLatest",
		"limit": limit,
	}).Debug("Fetching latest KuCoin orders")

	var orders []model.KucoinOrder
	err := r.db.WithContext(ctx).
		Order("id DESC").
		Limit(limit).
		Find(&orders).Error
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":  "KucoinOrderRepository",
			"op":    "FindLatest",
			"limit": limit,
		}).WithError(err).Error("Failed to fetch latest KuCoin orders")
		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "KucoinOrderRepository",
		"op":          "FindLatest",
		"limit":       limit,
		"rows_return": len(orders),
	}).Info("Latest KuCoin orders fetched")

	return orders, nil
}

// FindLatestBySymbol returns the latest KuCoin orders for a given symbol.
func (r *KucoinOrderRepository) FindLatestBySymbol(ctx context.Context, symbol string, limit int) ([]model.KucoinOrder, error) {
	if limit <= 0 {
		limit = 20
	}

	logger.WithFields(map[string]interface{}{
		"repo":   "KucoinOrderRepository",
		"op":     "FindLatestBySymbol",
		"symbol": symbol,
		"limit":  limit,
	}).Debug("Fetching latest KuCoin orders by symbol")

	var orders []model.KucoinOrder
	err := r.db.WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("id DESC").
		Limit(limit).
		Find(&orders).Error
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":   "KucoinOrderRepository",
			"op":     "FindLatestBySymbol",
			"symbol": symbol,
			"limit":  limit,
		}).WithError(err).Error("Failed to fetch latest KuCoin orders by symbol")
		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "KucoinOrderRepository",
		"op":          "FindLatestBySymbol",
		"symbol":      symbol,
		"limit":       limit,
		"rows_return": len(orders),
	}).Info("Latest KuCoin orders by symbol fetched")

	return orders, nil
}
