package repository

import (
	"context"
	"errors"
	"time"

	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"strategyexecutor/src/database"
	"strategyexecutor/src/model"
)

// OrderRepository handles read/write operations for orders and their execution logs.
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new repository instance using the main read/write database.
func NewOrderRepository() *OrderRepository {
	logger.WithField("component", "OrderRepository").
		Info("Creating new OrderRepository with MainDB")

	return &OrderRepository{
		db: database.MainDB,
	}
}

// WithDB allows overriding the underlying *gorm.DB instance.
// Useful for tests or when using a specific session/transaction.
func (r *OrderRepository) WithDB(db *gorm.DB) *OrderRepository {
	logger.WithField("component", "OrderRepository").
		Debug("Creating OrderRepository with custom DB instance")

	return &OrderRepository{db: db}
}

// ---------------------------------------------------
// Order methods
// ---------------------------------------------------

// Create inserts a new order into the database.
// The given order will be updated with the generated ID and timestamps.
func (r *OrderRepository) Create(
	ctx context.Context,
	order *model.Order,
) error {

	logger.WithFields(map[string]interface{}{
		"repo":   "OrderRepository",
		"op":     "Create",
		"symbol": order.Symbol,
		"side":   order.Side,
		"qty":    order.Quantity,
	}).Debug("Creating new order")

	err := r.db.WithContext(ctx).Create(order).Error
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo": "OrderRepository",
			"op":   "Create",
		}).WithError(err).Error("Failed to create order")

		return err
	}

	logger.WithFields(map[string]interface{}{
		"repo":     "OrderRepository",
		"op":       "Create",
		"order_id": order.ID,
	}).Info("Order created successfully")

	return nil
}

// FindByID fetches a single order by its primary ID.
// Returns (nil, nil) if the order is not found.
func (r *OrderRepository) FindByID(
	ctx context.Context,
	id uint,
) (*model.Order, error) {

	logger.WithFields(map[string]interface{}{
		"repo": "OrderRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("Fetching order by ID")

	var order model.Order

	err := r.db.WithContext(ctx).
		Preload("ExecutionLogs").
		First(&order, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo": "OrderRepository",
				"op":   "FindByID",
				"id":   id,
			}).Info("Order not found")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo": "OrderRepository",
			"op":   "FindByID",
			"id":   id,
		}).WithError(err).Error("Failed to fetch order by ID")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo": "OrderRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("Order fetched successfully")

	return &order, nil
}

// FindLatest returns the latest orders ordered from newest to oldest.
func (r *OrderRepository) FindLatest(
	ctx context.Context,
	limit int,
) ([]model.Order, error) {

	if limit <= 0 {
		limit = 20
	}

	logger.WithFields(map[string]interface{}{
		"repo":  "OrderRepository",
		"op":    "FindLatest",
		"limit": limit,
	}).Debug("Fetching latest orders")

	var orders []model.Order

	err := r.db.WithContext(ctx).
		Order("id DESC").
		Limit(limit).
		Find(&orders).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":  "OrderRepository",
			"op":    "FindLatest",
			"limit": limit,
		}).WithError(err).Error("Failed to fetch latest orders")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindLatest",
		"limit":       limit,
		"rows_return": len(orders),
	}).Info("Latest orders fetched")

	return orders, nil
}

// FindByExternalID fetches an order by its ExternalID.
// Returns (nil, nil) if the order is not found.
func (r *OrderRepository) FindByExternalID(
	ctx context.Context,
	externalID uint,
) (*model.Order, error) {

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindByExternalID",
		"external_id": externalID,
	}).Debug("Fetching order by external ID")

	var order model.Order

	err := r.db.WithContext(ctx).
		Where("external_id = ?", externalID).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo":        "OrderRepository",
				"op":          "FindByExternalID",
				"external_id": externalID,
			}).Info("Order not found by external ID")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo":        "OrderRepository",
			"op":          "FindByExternalID",
			"external_id": externalID,
		}).WithError(err).Error("Failed to fetch order by external ID")

		return nil, err
	}

	return &order, nil
}

// FindByExternalIDAndUser fetches an order by its ExternalID and UserStrID.
// Returns (nil, nil) if the order is not found.
func (r *OrderRepository) FindByExternalIDAndUser(
	ctx context.Context,
	userID string,
	externalID uint,
) (*model.Order, error) {

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindByExternalIDAndUser",
		"user_id":     userID,
		"external_id": externalID,
	}).Debug("Fetching order by external ID and user")

	var order model.Order

	err := r.db.WithContext(ctx).
		Where("external_id = ? AND user_id = ?", externalID, userID).
		Order("created_at DESC").
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo":        "OrderRepository",
				"op":          "FindByExternalIDAndUser",
				"user_id":     userID,
				"external_id": externalID,
			}).Info("Order not found by external ID and user")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo":        "OrderRepository",
			"op":          "FindByExternalIDAndUser",
			"user_id":     userID,
			"external_id": externalID,
		}).WithError(err).Error("Failed to fetch order by external ID and user")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindByExternalIDAndUser",
		"user_id":     userID,
		"external_id": externalID,
		"order_id":    order.ID,
	}).Debug("Order fetched successfully by external ID and user")

	return &order, nil
}

func (r *OrderRepository) FindByExternalIDAndUserID(
	ctx context.Context,
	userID uint,
	externalID uint,
	orderDir string,
) (*model.Order, error) {

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindByExternalIDAndUser",
		"user_id":     userID,
		"external_id": externalID,
	}).Debug("Fetching order by external ID and user")

	var order model.Order

	err := r.db.WithContext(ctx).
		Where("external_id = ? AND user_id = ? and order_dir = ?", externalID, userID, orderDir).
		Order("created_at DESC").
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo":        "OrderRepository",
				"op":          "FindByExternalIDAndUser",
				"user_id":     userID,
				"external_id": externalID,
			}).Info("Order not found by external ID and user")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo":        "OrderRepository",
			"op":          "FindByExternalIDAndUser",
			"user_id":     userID,
			"external_id": externalID,
		}).WithError(err).Error("Failed to fetch order by external ID and user")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindByExternalIDAndUser",
		"user_id":     userID,
		"external_id": externalID,
		"order_id":    order.ID,
	}).Debug("Order fetched successfully by external ID and user")

	return &order, nil
}

// UpdateStatus updates only the status of the given order ID.
func (r *OrderRepository) UpdateStatus(
	ctx context.Context,
	id uint,
	status string,
) error {

	logger.WithFields(map[string]interface{}{
		"repo":   "OrderRepository",
		"op":     "UpdateStatus",
		"id":     id,
		"status": status,
	}).Debug("Updating order status")

	err := r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Update("status", status).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":   "OrderRepository",
			"op":     "UpdateStatus",
			"id":     id,
			"status": status,
		}).WithError(err).Error("Failed to update order status")

		return err
	}

	logger.WithFields(map[string]interface{}{
		"repo":   "OrderRepository",
		"op":     "UpdateStatus",
		"id":     id,
		"status": status,
	}).Info("Order status updated successfully")

	return nil
}

// UpdateStopLoss updates only the SL of the given order ID.
func (r *OrderRepository) UpdateStopLoss(
	ctx context.Context,
	id uint,
	stopLoss float64,
) error {

	logger.WithFields(map[string]interface{}{
		"repo":          "OrderRepository",
		"op":            "UpdateStatus",
		"id":            id,
		"stop_loss_pct": stopLoss,
	}).Debug("Updating order SL")

	err := r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", id).
		Update("stop_loss_pct", stopLoss).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":          "OrderRepository",
			"op":            "UpdateStatus",
			"id":            id,
			"stop_loss_pct": stopLoss,
		}).WithError(err).Error("Failed to update order status")

		return err
	}

	logger.WithFields(map[string]interface{}{
		"repo":          "OrderRepository",
		"op":            "UpdateStatus",
		"id":            id,
		"stop_loss_pct": stopLoss,
	}).Info("Order stop_loss updated successfully")

	return nil
}

// ---------------------------------------------------
// OrderExecutionLog methods
// ---------------------------------------------------

func (r *OrderRepository) CreateExecutionLog(
	ctx context.Context,
	logEntry *model.OrderExecutionLog,
) error {

	logger.WithFields(map[string]interface{}{
		"repo":     "OrderRepository",
		"op":       "CreateExecutionLog",
		"order_id": logEntry.OrderID,
		"status":   logEntry.Status,
	}).Debug("Creating execution log")

	err := r.db.WithContext(ctx).Create(logEntry).Error
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":     "OrderRepository",
			"op":       "CreateExecutionLog",
			"order_id": logEntry.OrderID,
		}).WithError(err).Error("Failed to create execution log")

		return err
	}

	logger.WithFields(map[string]interface{}{
		"repo":     "OrderRepository",
		"op":       "CreateExecutionLog",
		"order_id": logEntry.OrderID,
	}).Info("Execution log created")

	return nil
}

func (r *OrderRepository) FindExecutionLogsByOrderID(
	ctx context.Context,
	orderID uint,
) ([]model.OrderExecutionLog, error) {

	logger.WithFields(map[string]interface{}{
		"repo":     "OrderRepository",
		"op":       "FindExecutionLogsByOrderID",
		"order_id": orderID,
	}).Debug("Fetching execution logs for order")

	var logs []model.OrderExecutionLog

	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("id ASC").
		Find(&logs).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":     "OrderRepository",
			"op":       "FindExecutionLogsByOrderID",
			"order_id": orderID,
		}).WithError(err).Error("Failed to fetch execution logs")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindExecutionLogsByOrderID",
		"order_id":    orderID,
		"rows_return": len(logs),
	}).Info("Execution logs fetched")

	return logs, nil
}

func (r *OrderRepository) FindLastExecutionLogByOrderID(
	ctx context.Context,
	orderID uint,
) (*model.OrderExecutionLog, error) {

	logger.WithFields(map[string]interface{}{
		"repo":     "OrderRepository",
		"op":       "FindLastExecutionLogByOrderID",
		"order_id": orderID,
	}).Debug("Fetching last execution log")

	var logEntry model.OrderExecutionLog

	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("id DESC").
		First(&logEntry).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo":     "OrderRepository",
				"op":       "FindLastExecutionLogByOrderID",
				"order_id": orderID,
			}).Info("No execution log found for order")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo":     "OrderRepository",
			"op":       "FindLastExecutionLogByOrderID",
			"order_id": orderID,
		}).WithError(err).Error("Failed to fetch last execution log")

		return nil, err
	}

	return &logEntry, nil
}

// ---------------------------------------------------
// Transaction helpers
// ---------------------------------------------------

func (r *OrderRepository) CreateWithAutoLog(
	ctx context.Context,
	order *model.Order,
) error {

	logger.WithFields(map[string]interface{}{
		"repo":   "OrderRepository",
		"op":     "CreateWithAutoLog",
		"symbol": order.Symbol,
		"side":   order.Side,
	}).Info("Creating order with automatic execution log")

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			logger.WithError(err).Error("Failed to create order inside transaction")
			return err
		}

		logEntry := &model.OrderLog{
			OrderID:       order.ID,
			ExchangeID:    order.ExchangeID,
			Symbol:        order.Symbol,
			Side:          order.Side,
			PosSide:       order.PosSide,
			OrderType:     order.OrderType,
			Quantity:      order.Quantity,
			Price:         order.Price,
			StopLossPct:   order.StopLossPct,
			TakeProfitPct: order.TakeProfitPct,
			Status:        order.Status,
			CreatedAt:     time.Now(),
			//OrderDir:      order.OrderDir,
		}

		if err := tx.Create(logEntry).Error; err != nil {
			logger.WithError(err).Error("Failed to create auto execution log")
			return err
		}

		return nil
	})
}

func (r *OrderRepository) UpdateStatusWithAutoLog(
	ctx context.Context,
	orderID uint,
	newStatus string,
	reason string,
) error {

	logger.WithFields(map[string]interface{}{
		"repo":      "OrderRepository",
		"op":        "UpdateStatusWithAutoLog",
		"order_id":  orderID,
		"newStatus": newStatus,
		"reason":    reason,
	}).Info("Updating order status with automatic execution log")

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order

		if err := tx.First(&order, orderID).Error; err != nil {
			logger.WithError(err).Error("Failed to load order inside transaction")
			return err
		}

		if err := tx.
			Model(&model.Order{}).
			Where("id = ?", orderID).
			Update("status", newStatus).Error; err != nil {
			logger.WithError(err).Error("Failed to update order status inside transaction")
			return err
		}

		logEntry := &model.OrderLog{
			OrderID:       order.ID,
			ExchangeID:    order.ExchangeID,
			Symbol:        order.Symbol,
			Side:          order.Side,
			PosSide:       order.PosSide,
			OrderType:     order.OrderType,
			Quantity:      order.Quantity,
			Price:         order.Price,
			StopLossPct:   order.StopLossPct,
			TakeProfitPct: order.TakeProfitPct,
			Status:        newStatus,
			CreatedAt:     time.Now(),
		}

		if err := tx.Create(logEntry).Error; err != nil {
			logger.WithError(err).Error("Failed to create auto execution log on status update")
			return err
		}

		return nil
	})
}

func (r *OrderRepository) UpdatePriceAutoLog(
	ctx context.Context,
	orderID uint,
	price *float64,
	reason string,
) error {

	logger.WithFields(map[string]interface{}{
		"repo":     "OrderRepository",
		"op":       "UpdateStatusWithAutoLog",
		"order_id": orderID,
		"price":    price,
		"reason":   reason,
	}).Info("Updating order price with automatic execution log")

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order

		if err := tx.First(&order, orderID).Error; err != nil {
			logger.WithError(err).Error("Failed to load order inside transaction")
			return err
		}

		if err := tx.
			Model(&model.Order{}).
			Where("id = ?", orderID).
			Update("price", price).Error; err != nil {
			logger.WithError(err).Error("Failed to update order status inside transaction")
			return err
		}

		logEntry := &model.OrderLog{
			OrderID:       order.ID,
			ExchangeID:    order.ExchangeID,
			Symbol:        order.Symbol,
			Side:          order.Side,
			PosSide:       order.PosSide,
			OrderType:     order.OrderType,
			Quantity:      order.Quantity,
			Price:         price,
			StopLossPct:   order.StopLossPct,
			TakeProfitPct: order.TakeProfitPct,
			Status:        order.Status,
			CreatedAt:     time.Now(),
		}

		if err := tx.Create(logEntry).Error; err != nil {
			logger.WithError(err).Error("Failed to create auto execution log on status update")
			return err
		}

		return nil
	})
}

// FindByExchangeIDAndUserID fetches an order by its ExchangeID and UserStrID.
// Returns (nil, nil) if the order is not found.
func (r *OrderRepository) FindByExchangeIDAndUserID(
	ctx context.Context,
	userID uint,
	exchangeID uint,
) (*model.Order, error) {

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindByExternalIDAndUser",
		"user_id":     userID,
		"exchange_id": exchangeID,
	}).Debug("Fetching order by external ID and user")

	var order model.Order

	err := r.db.WithContext(ctx).
		Where("exchange_id = ? AND user_id = ?", exchangeID, userID).
		Order("created_at DESC").
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo":        "OrderRepository",
				"op":          "FindByExternalIDAndUser",
				"user_id":     userID,
				"exchange_id": exchangeID,
			}).Info("Order not found by external ID and user")

			return nil, nil
		}

		logger.WithFields(map[string]interface{}{
			"repo":        "OrderRepository",
			"op":          "FindByExternalIDAndUser",
			"user_id":     userID,
			"exchange_id": exchangeID,
		}).WithError(err).Error("Failed to fetch order by external ID and user")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "OrderRepository",
		"op":          "FindByExternalIDAndUser",
		"user_id":     userID,
		"exchange_id": exchangeID,
		"order_id":    order.ID,
	}).Debug("Order fetched successfully by external ID and user")

	return &order, nil
}
