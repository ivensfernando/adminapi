// repository/trading_signal_repository.go
package repository

import (
	"context"
	"errors"

	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"adminapi/src/database"      // TODO: adjust to your real module path
	"adminapi/src/externalmodel" // TODO: adjust to your real module path
)

// TradingSignalRepository handles read-only operations
// for external trading signals stored in the read-only database.
type TradingSignalRepository struct {
	db *gorm.DB
}

// NewTradingSignalRepository creates a new repository instance.
// It uses the ReadOnlyDB connection by default.
func NewTradingSignalRepository() *TradingSignalRepository {
	logger.WithField("component", "TradingSignalRepository").
		Info("Creating new TradingSignalRepository with ReadOnlyDB")

	return &TradingSignalRepository{
		db: database.ReadOnlyDB,
	}
}

// WithDB allows overriding the underlying *gorm.DB instance.
// Useful for tests or custom sessions/transactions (even if read-only).
func (r *TradingSignalRepository) WithDB(db *gorm.DB) *TradingSignalRepository {
	logger.WithField("component", "TradingSignalRepository").
		Debug("Creating new TradingSignalRepository with custom DB instance")

	return &TradingSignalRepository{db: db}
}

// FindByID fetches a single trading signal by its primary ID.
// Returns (nil, nil) if not found.
func (r *TradingSignalRepository) FindByID(
	ctx context.Context,
	id uint,
) (*externalmodel.TradingSignal, error) {

	logger.WithFields(map[string]interface{}{
		"repo": "TradingSignalRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("Fetching trading signal by ID")

	var signal externalmodel.TradingSignal

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&signal).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WithFields(map[string]interface{}{
				"repo": "TradingSignalRepository",
				"op":   "FindByID",
				"id":   id,
			}).Info("Trading signal not found")
			return nil, nil // not found is not an error
		}

		logger.WithFields(map[string]interface{}{
			"repo": "TradingSignalRepository",
			"op":   "FindByID",
			"id":   id,
		}).WithError(err).Error("Failed to fetch trading signal by ID")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo": "TradingSignalRepository",
		"op":   "FindByID",
		"id":   id,
	}).Debug("Trading signal fetched successfully")

	return &signal, nil
}

// FindLatest fetches the latest trading signals ordered from newest to oldest.
// The limit parameter defines how many records will be returned.
func (r *TradingSignalRepository) FindLatest(
	ctx context.Context,
	limit int,
) ([]externalmodel.TradingSignal, error) {

	if limit <= 0 {
		limit = 10 // default safety limit
	}

	logger.WithFields(map[string]interface{}{
		"repo":  "TradingSignalRepository",
		"op":    "FindLatest",
		"limit": limit,
	}).Debug("Fetching latest trading signals")

	var signals []externalmodel.TradingSignal

	err := r.db.WithContext(ctx).
		Select("id", "order_id", "symbol", "action", "price").
		Order("id DESC").
		Limit(limit).
		Find(&signals).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":  "TradingSignalRepository",
			"op":    "FindLatest",
			"limit": limit,
		}).WithError(err).Error("Failed to fetch latest trading signals")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "TradingSignalRepository",
		"op":          "FindLatest",
		"limit":       limit,
		"rows_return": len(signals),
	}).Info("Latest trading signals fetched")

	return signals, nil
}

// FindAfterID fetches trading signals with ID greater than lastID,
// ordered from oldest to newest (ascending by ID).
// This is ideal for incremental polling every N seconds.
func (r *TradingSignalRepository) FindAfterID(
	ctx context.Context,
	lastID uint,
	limit int,
) ([]externalmodel.TradingSignal, error) {

	if limit <= 0 {
		limit = 100 // default safety limit
	}

	logger.WithFields(map[string]interface{}{
		"repo":   "TradingSignalRepository",
		"op":     "FindAfterID",
		"lastID": lastID,
		"limit":  limit,
	}).Debug("Fetching trading signals after ID")

	var signals []externalmodel.TradingSignal

	err := r.db.WithContext(ctx).
		Where("id > ?", lastID).
		Order("id ASC").
		Limit(limit).
		Find(&signals).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":   "TradingSignalRepository",
			"op":     "FindAfterID",
			"lastID": lastID,
			"limit":  limit,
		}).WithError(err).Error("Failed to fetch trading signals after ID")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "TradingSignalRepository",
		"op":          "FindAfterID",
		"lastID":      lastID,
		"limit":       limit,
		"rows_return": len(signals),
	}).Info("Trading signals after ID fetched")

	return signals, nil
}

// FindBetweenIDs fetches trading signals with ID in the interval (startID, endID],
// ordered from oldest to newest (ascending by ID).
func (r *TradingSignalRepository) FindBetweenIDs(
	ctx context.Context,
	startID, endID uint,
	limit int,
) ([]externalmodel.TradingSignal, error) {

	if limit <= 0 {
		limit = 100
	}

	logger.WithFields(map[string]interface{}{
		"repo":    "TradingSignalRepository",
		"op":      "FindBetweenIDs",
		"startID": startID,
		"endID":   endID,
		"limit":   limit,
	}).Debug("Fetching trading signals between IDs")

	var signals []externalmodel.TradingSignal

	err := r.db.WithContext(ctx).
		Where("id > ? AND id <= ?", startID, endID).
		Order("id ASC").
		Limit(limit).
		Find(&signals).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":    "TradingSignalRepository",
			"op":      "FindBetweenIDs",
			"startID": startID,
			"endID":   endID,
			"limit":   limit,
		}).WithError(err).Error("Failed to fetch trading signals between IDs")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "TradingSignalRepository",
		"op":          "FindBetweenIDs",
		"startID":     startID,
		"endID":       endID,
		"limit":       limit,
		"rows_return": len(signals),
	}).Info("Trading signals between IDs fetched")

	return signals, nil
}

// FindBySymbol fetches the latest trading signals for a given symbol,
// ordered from newest to oldest.
func (r *TradingSignalRepository) FindBySymbol(
	ctx context.Context,
	symbol string,
	limit int,
) ([]externalmodel.TradingSignal, error) {

	if limit <= 0 {
		limit = 50
	}

	logger.WithFields(map[string]interface{}{
		"repo":   "TradingSignalRepository",
		"op":     "FindBySymbol",
		"symbol": symbol,
		"limit":  limit,
	}).Debug("Fetching trading signals by symbol")

	var signals []externalmodel.TradingSignal

	err := r.db.WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("id DESC").
		Limit(limit).
		Find(&signals).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":   "TradingSignalRepository",
			"op":     "FindBySymbol",
			"symbol": symbol,
			"limit":  limit,
		}).WithError(err).Error("Failed to fetch trading signals by symbol")

		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":        "TradingSignalRepository",
		"op":          "FindBySymbol",
		"symbol":      symbol,
		"limit":       limit,
		"rows_return": len(signals),
	}).Info("Trading signals by symbol fetched")

	return signals, nil
}

// CountNewAfterID returns how many new records exist with ID greater than lastID.
// This can be used to quickly check if there is new data before doing a heavier fetch.
func (r *TradingSignalRepository) CountNewAfterID(
	ctx context.Context,
	lastID uint,
) (int64, error) {

	logger.WithFields(map[string]interface{}{
		"repo":   "TradingSignalRepository",
		"op":     "CountNewAfterID",
		"lastID": lastID,
	}).Debug("Counting new trading signals after ID")

	var count int64

	err := r.db.WithContext(ctx).
		Model(&externalmodel.TradingSignal{}).
		Where("id > ?", lastID).
		Count(&count).Error

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"repo":   "TradingSignalRepository",
			"op":     "CountNewAfterID",
			"lastID": lastID,
		}).WithError(err).Error("Failed to count new trading signals after ID")

		return 0, err
	}

	logger.WithFields(map[string]interface{}{
		"repo":   "TradingSignalRepository",
		"op":     "CountNewAfterID",
		"lastID": lastID,
		"count":  count,
	}).Info("Counted new trading signals after ID")

	return count, nil
}
