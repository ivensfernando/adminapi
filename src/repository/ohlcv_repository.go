package repository

import (
	"context"
	"errors"
	"strategyexecutor/src/database"
	"strategyexecutor/src/model"
	"strategyexecutor/src/tp_sl"
	"time"

	"github.com/shopspring/decimal"
	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var ErrInvalidInterval = errors.New("invalid interval. allowed: 5m,15m,30m,45m")

type OHLCVRepository struct {
	db *gorm.DB
}

// NewOHLCVRepositoryRepository creates a new repository using the given gorm DB.
func NewOHLCVRepositoryRepository() *OHLCVRepository {
	logger.WithField("component", "ExchangeRepository").
		Info("Creating new ExchangeRepository with custom DB instance")

	return &OHLCVRepository{
		db: database.MainDB,
	}
}

func NewOHLCVRepositoryRepositoryWithDB(db *gorm.DB) *OHLCVRepository {
	logger.WithField("component", "ExchangeRepository").
		Info("Creating new ExchangeRepository with custom DB instance")

	return &OHLCVRepository{
		db: db,
	}
}

func (s *OHLCVRepository) FetchRecentOHLCV1m(
	ctx context.Context,
	symbol string,
	to time.Time,
	limit int,
) ([]model.OHLCVCrypto1m, error) {
	if limit <= 0 {
		limit = 200
	}

	var rows []model.OHLCVCrypto1m
	err := s.db.WithContext(ctx).
		Where("symbol = ? AND datetime <= ?", symbol, to).
		Order("datetime DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	// reverse to ascending chronological order for easier logic
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}
func (s *OHLCVRepository) GetNextStopLoss(
	ctx context.Context,
	symbol string,
	now time.Time,
	side tp_sl.Side, // long or short
	currentSL decimal.Decimal,
	interval time.Duration, // 1m, 5m, 15m, 30m, 45m
	lookback int, // e.g. 20
) (decimal.Decimal, bool, error) {

	if lookback <= 0 {
		lookback = 20
	}

	// Fetch enough 1m candles to build lookback aggregated candles.
	// Need at least lookback + 2 aggregated candles because SL logic reads "previous candle".
	mult := int(interval.Minutes())
	if mult <= 0 {
		mult = 1
	}

	needAgg := lookback + 2
	limit1m := needAgg*mult + (2 * mult)

	candles1m, err := s.FetchRecentOHLCV1m(ctx, symbol, now, limit1m)
	if err != nil {
		return decimal.Zero, false, err
	}

	candles := candles1m
	if interval > time.Minute {
		agg, err := AggregateOHLCVFrom1m(candles1m, interval)
		if err != nil {
			return decimal.Zero, false, err
		}
		candles = agg
	}

	// Ensure we have enough candles for SL logic
	if len(candles) < 2 {
		return currentSL, false, nil
	}

	// Keep only what we need. SL uses last lookback candles.
	if len(candles) > needAgg {
		candles = candles[len(candles)-needAgg:]
	}

	newSL, moved := tp_sl.ComputeNextStopLossDirectional(side, currentSL, candles, lookback)
	return newSL, moved, nil
}

func bucketStart(t time.Time, interval time.Duration) time.Time {
	// Works for intervals that are multiples of 1 minute
	// Align to wall-clock boundaries: 12:07 with 5m => 12:05
	secs := t.Unix()
	step := int64(interval.Seconds())
	return time.Unix((secs/step)*step, 0).UTC()
}

func AggregateOHLCVFrom1m(
	candles []model.OHLCVCrypto1m,
	interval time.Duration,
) ([]model.OHLCVCrypto1m, error) {
	if interval != 5*time.Minute &&
		interval != 15*time.Minute &&
		interval != 30*time.Minute &&
		interval != 45*time.Minute {
		return nil, ErrInvalidInterval
	}

	if len(candles) == 0 {
		return []model.OHLCVCrypto1m{}, nil
	}

	out := make([]model.OHLCVCrypto1m, 0, len(candles)/int(interval.Minutes())+2)

	var cur model.OHLCVCrypto1m
	var curBucket time.Time
	hasCur := false

	for _, c := range candles {
		b := bucketStart(c.Datetime, interval)

		if !hasCur || !b.Equal(curBucket) {
			// flush previous bucket
			if hasCur {
				out = append(out, cur)
			}
			// start new bucket
			curBucket = b
			hasCur = true
			cur = model.OHLCVCrypto1m{
				Symbol:   c.Symbol,
				Datetime: curBucket, // bucket open time
				Open:     c.Open,
				High:     c.High,
				Low:      c.Low,
				Close:    c.Close,
				Volume:   c.Volume,
			}
			continue
		}

		// aggregate
		if c.High.GreaterThan(cur.High) {
			cur.High = c.High
		}
		if c.Low.LessThan(cur.Low) {
			cur.Low = c.Low
		}
		cur.Close = c.Close
		cur.Volume = cur.Volume.Add(c.Volume)
	}

	if hasCur {
		out = append(out, cur)
	}

	return out, nil
}

func (s *OHLCVRepository) FetchRecentOHLCVAgg(
	ctx context.Context,
	symbol string,
	to time.Time,
	interval time.Duration,
	limitAgg int,
) ([]model.OHLCVCrypto1m, error) {
	if limitAgg <= 0 {
		limitAgg = 200
	}

	// fetch enough 1m candles to build limitAgg aggregated candles
	mult := int(interval.Minutes())
	if mult <= 0 {
		return nil, ErrInvalidInterval
	}
	limit1m := limitAgg*mult + mult // small buffer

	rows1m, err := s.FetchRecentOHLCV1m(ctx, symbol, to, limit1m)
	if err != nil {
		return nil, err
	}

	agg, err := AggregateOHLCVFrom1m(rows1m, interval)
	if err != nil {
		return nil, err
	}

	// keep only the most recent limitAgg (agg is ascending)
	if len(agg) > limitAgg {
		agg = agg[len(agg)-limitAgg:]
	}
	return agg, nil
}
