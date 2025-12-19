package tp_sl_test

import (
	"context"
	"regexp"
	"strategyexecutor/src/repository"
	"strategyexecutor/src/tp_sl"
	"testing"
	"time"

	"strategyexecutor/src/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func setupDBMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	require.NoError(t, err)

	return gormDB, mock
}

func mustNYorUTC(t *testing.T) *time.Location {
	loc, err := time.LoadLocation("UTC")
	require.NoError(t, err)
	return loc
}

func oneMinCandle(dt time.Time, o, h, l, c, v float64) model.OHLCVCrypto1m {
	return model.OHLCVCrypto1m{
		ID:       0,
		Symbol:   "BTCUSDT",
		Datetime: dt,
		Open:     decimal.NewFromFloat(o),
		High:     decimal.NewFromFloat(h),
		Low:      decimal.NewFromFloat(l),
		Close:    decimal.NewFromFloat(c),
		Volume:   decimal.NewFromFloat(v),
	}
}

// build15CandlesFor3x5mBuckets returns 15 candles (00:00..00:14),
// designed so the aggregated 5m candles are:
// bucket0 [00:00..00:04] open=100 close=101 low=99  high=102
// bucket1 [00:05..00:09] open=101 close=105 low=100 high=106 (bullish)
// bucket2 [00:10..00:14] open=105 close=104 low=103 high=107
//
// With lookback=2:
// floorAvgLow = avg(low(bucket1)=100, low(bucket2)=103) = 101.5
// prev candle (bucket1) bullish => allowed to raise
func build15CandlesFor3x5mBuckets(t *testing.T, start time.Time) []model.OHLCVCrypto1m {
	candles := make([]model.OHLCVCrypto1m, 0, 15)

	// bucket0
	for i := 0; i < 5; i++ {
		dt := start.Add(time.Duration(i) * time.Minute)
		open := 100.0
		close := 100.0
		if i == 0 {
			open = 100.0
		}
		if i == 4 {
			close = 101.0 // bucket close
		}
		candles = append(candles, oneMinCandle(dt, open, 102.0, 99.0, close, 1.0))
	}

	// bucket1 (bullish aggregated)
	for i := 5; i < 10; i++ {
		dt := start.Add(time.Duration(i) * time.Minute)
		open := 101.0
		close := 101.0
		if i == 5 {
			open = 101.0 // bucket open
		}
		if i == 9 {
			close = 105.0 // bucket close (bullish)
		}
		candles = append(candles, oneMinCandle(dt, open, 106.0, 100.0, close, 1.0))
	}

	// bucket2 (bearish aggregated)
	for i := 10; i < 15; i++ {
		dt := start.Add(time.Duration(i) * time.Minute)
		open := 105.0
		close := 105.0
		if i == 10 {
			open = 105.0 // bucket open
		}
		if i == 14 {
			close = 104.0 // bucket close (bearish)
		}
		candles = append(candles, oneMinCandle(dt, open, 107.0, 103.0, close, 1.0))
	}

	require.Len(t, candles, 15)
	return candles
}

func TestOHLCVRepository_GetNextStopLoss_Aggregated5m_UsesFloorAvgLow(t *testing.T) {
	db, mock := setupDBMock(t)

	repo := repository.NewOHLCVRepositoryRepositoryWithDB(db)

	loc := mustNYorUTC(t)
	start := time.Date(2025, 3, 1, 0, 0, 0, 0, loc)
	now := start.Add(14 * time.Minute) // last candle time

	// Build candles ascending for our reasoning
	candlesAsc := build15CandlesFor3x5mBuckets(t, start)

	// Your FetchRecentOHLCV1m does ORDER BY datetime DESC and then reverses.
	// So we must return rows in DESC order from sqlmock.
	rows := sqlmock.NewRows([]string{
		"id", "symbol", "datetime", "open", "high", "low", "close", "volume",
	})

	for i := len(candlesAsc) - 1; i >= 0; i-- {
		c := candlesAsc[i]
		rows.AddRow(
			uint(i+1),
			c.Symbol,
			c.Datetime,
			c.Open.InexactFloat64(),
			c.High.InexactFloat64(),
			c.Low.InexactFloat64(),
			c.Close.InexactFloat64(),
			c.Volume.InexactFloat64(),
		)
	}

	// Match the SELECT produced by GORM. Keep it flexible.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "ohlcv_crypto_1m" WHERE symbol = $1 AND datetime <= $2 ORDER BY datetime DESC LIMIT $3`)).
		WithArgs("BTCUSDT", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	currentSL := d("100.0")
	interval := 5 * time.Minute
	lookback := 2

	newSL, raised, err := repo.GetNextStopLoss(context.Background(), "BTCUSDT", now, tp_sl.SideLong, currentSL, interval, lookback)
	require.NoError(t, err)
	require.False(t, raised)

	expected := d("100.0")
	require.True(t, newSL.Equal(expected), "expected newSL=%s got=%s", expected.String(), newSL.String())

	require.NoError(t, mock.ExpectationsWereMet())
}

// build15CandlesFor3x5mBucketsShort returns 15 candles (00:00..00:14),
// designed so aggregated 5m candles are:
// bucket0 [00:00..00:04] anything
// bucket1 [00:05..00:09] open=110 close=100 high=115 (bearish)  => prev candle for gating
// bucket2 [00:10..00:14] open=100 close=98  high=112 (bearish/doesn't matter for gating)
//
// With lookback=2, window for ceilAvgHigh is (bucket1.High=115, bucket2.High=112) => avg = 113.5
// Clamp: candidate = max(ceilAvgHigh, prev.High) = max(113.5, 115) = 115
// If currentSL = 120, we should tighten down to 115 (moved=true).
func build15CandlesFor3x5mBucketsShort(t *testing.T, start time.Time) []model.OHLCVCrypto1m {
	candles := make([]model.OHLCVCrypto1m, 0, 15)

	// bucket0
	for i := 0; i < 5; i++ {
		dt := start.Add(time.Duration(i) * time.Minute)
		open := 100.0
		close := 100.0
		if i == 4 {
			close = 101.0
		}
		candles = append(candles, oneMinCandle(dt, open, 105.0, 95.0, close, 1.0))
	}

	// bucket1 (bearish aggregated) prev candle used by SL logic
	for i := 5; i < 10; i++ {
		dt := start.Add(time.Duration(i) * time.Minute)

		open := 110.0
		close := 110.0
		if i == 5 {
			open = 110.0 // bucket open
		}
		if i == 9 {
			close = 100.0 // bucket close, bearish (100 < 110)
		}

		// keep high fixed at 115 across the bucket
		candles = append(candles, oneMinCandle(dt, open, 115.0, 99.0, close, 1.0))
	}

	// bucket2 (latest bucket)
	for i := 10; i < 15; i++ {
		dt := start.Add(time.Duration(i) * time.Minute)

		open := 100.0
		close := 100.0
		if i == 10 {
			open = 100.0
		}
		if i == 14 {
			close = 98.0
		}

		// high fixed at 112
		candles = append(candles, oneMinCandle(dt, open, 112.0, 95.0, close, 1.0))
	}

	require.Len(t, candles, 15)
	return candles
}

func TestOHLCVRepository_GetNextStopLoss_Aggregated5m_Short_UsesCeilAvgHigh(t *testing.T) {
	db, mock := setupDBMock(t)
	repo := repository.NewOHLCVRepositoryRepositoryWithDB(db)

	loc := mustNYorUTC(t)
	start := time.Date(2025, 3, 1, 0, 0, 0, 0, loc)
	now := start.Add(14 * time.Minute)

	candlesAsc := build15CandlesFor3x5mBucketsShort(t, start)

	rows := sqlmock.NewRows([]string{
		"id", "symbol", "datetime", "open", "high", "low", "close", "volume",
	})

	// return DESC order (repo reverses)
	for i := len(candlesAsc) - 1; i >= 0; i-- {
		c := candlesAsc[i]
		rows.AddRow(
			uint(i+1),
			c.Symbol,
			c.Datetime,
			c.Open.InexactFloat64(),
			c.High.InexactFloat64(),
			c.Low.InexactFloat64(),
			c.Close.InexactFloat64(),
			c.Volume.InexactFloat64(),
		)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "ohlcv_crypto_1m" WHERE symbol = $1 AND datetime <= $2 ORDER BY datetime DESC LIMIT $3`)).
		WithArgs("BTCUSDT", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	currentSL := d("120.0")
	interval := 5 * time.Minute
	lookback := 2

	newSL, moved, err := repo.GetNextStopLoss(context.Background(), "BTCUSDT", now, tp_sl.SideShort, currentSL, interval, lookback)
	require.NoError(t, err)
	require.True(t, moved)

	// As explained above, candidate clamps to prev.High=115
	expected := d("115.0")
	require.True(t, newSL.Equal(expected), "expected newSL=%s got=%s", expected.String(), newSL.String())

	require.NoError(t, mock.ExpectationsWereMet())
}
