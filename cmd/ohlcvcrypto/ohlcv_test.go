package ohlcvcrypto

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strategyexecutor/src/model"
	"strategyexecutor/src/utils"

	"testing"
	"time"

	"github.com/nntaoli-project/goex/binance"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nntaoli-project/goex"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupDBMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	require.NoError(t, err)

	return gormDB, mock
}

func setupMockBinanceServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/v3/klines", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Sample JSON response directly from Binance API documentation or captured API responses
		_, err := w.Write([]byte(`[
			[1499040000000, "0.01634790", "0.80000000", "0.01575800", "0.01577100", "148976.11427815", 1499644799999, "2434.19055334", 308, "1756.87402397", "28.46694368", "17928899.62484339"]
		]`))
		if err != nil {
			return
		}
	})
	return httptest.NewServer(handler)
}

func TestOHLCVCrypto_fetchOHLCVSeries(t *testing.T) {
	server := setupMockBinanceServer()
	defer server.Close()

	// Redirect API calls to the mock server
	apiConfig := &goex.APIConfig{
		HttpClient: http.DefaultClient,
		Endpoint:   server.URL, // Use mock server URL
	}

	db, _ := setupDBMock(t)
	ohlcv := OHLCVCrypto{
		DB: db,
		Config: &Config{
			Symbol:      "BTC_USDT",
			StartDt:     time.Now().Add(-24 * time.Hour),
			EndDt:       time.Now(),
			DurationStr: Duration1h,
		},
		exchange: binance.NewWithConfig(apiConfig),
	}

	klines, err := ohlcv.fetchOHLCVSeries()
	require.NoError(t, err)
	require.Len(t, klines, 1, "Should fetch exactly one OHLCV record")
	require.InDelta(t, 0.01634790, klines[0].Open, 0, "Open price should match")
}

// Test determineStartPoint for valid start point retrieval.
func TestOHLCVCrypto_determineStartPoint(t *testing.T) {
	db, mock := setupDBMock(t)

	config := &Config{
		DurationStr: "1h",
		StartDt:     utils.ResetTime(time.Now().Add(-24*time.Hour), "minute"),
		EndDt:       time.Now(),
	}

	ohlcv := OHLCVCrypto{
		Log:    logrus.NewEntry(logrus.New()),
		DB:     db,
		Config: config,
	}
	ohlcv.exchange = ohlcv.newBinanceInstance()

	mock.ExpectQuery(`SELECT MAX\(datetime\)`).WillReturnRows(sqlmock.NewRows([]string{"MAX(datetime)"}).
		AddRow(sql.NullTime{Time: utils.ResetTime(time.Now().Add(-time.Hour), "minute"), Valid: true}))

	err := ohlcv.determineStartPoint()
	require.NoError(t, err, "Expected determineStartPoint to complete without error")
	require.Equal(t, utils.ResetTime(time.Now().Add(-2*time.Hour), "minute").String(), config.StartDt.String(), "Start date should be adjusted based on last datetime")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test parseDuration for valid duration parsing based on config.
func TestOHLCVCrypto_parseDuration(t *testing.T) {
	tests := []struct {
		durationStr string
		expected    time.Duration
		shouldPanic bool
	}{
		{"1m", time.Minute, false},
		{"1h", time.Hour, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.durationStr, func(t *testing.T) {
			config := &Config{DurationStr: tt.durationStr}
			ohlcv := OHLCVCrypto{Config: config}

			if tt.shouldPanic {
				require.Panics(t, func() { _ = ohlcv.parseDuration() })
			} else {
				require.Equal(t, tt.expected, ohlcv.parseDuration())
			}
		})
	}
}

// Test parseDurationToGoex to verify translation to goex KlinePeriod.
func TestOHLCVCrypto_parseDurationToGoex(t *testing.T) {
	tests := []struct {
		durationStr string
		expected    goex.KlinePeriod
		shouldPanic bool
	}{
		{"1m", goex.KLINE_PERIOD_1MIN, false},
		{"1h", goex.KLINE_PERIOD_1H, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.durationStr, func(t *testing.T) {
			config := &Config{DurationStr: tt.durationStr}
			ohlcv := OHLCVCrypto{Config: config}

			if tt.shouldPanic {
				require.Panics(t, func() { _ = ohlcv.parseDurationToGoex() })
			} else {
				require.Equal(t, tt.expected, ohlcv.parseDurationToGoex())
			}
		})
	}
}

// Test getModel to verify correct model is chosen based on duration.
func TestOHLCVCrypto_getModel(t *testing.T) {
	db, _ := setupDBMock(t)

	tests := []struct {
		durationStr string
		expected    interface{}
		shouldPanic bool
	}{
		{"1m", &model.OHLCVCrypto1m{}, false},
		{"1h", &model.OHLCVCrypto1h{}, false},
		{"invalid", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.durationStr, func(t *testing.T) {
			config := &Config{DurationStr: tt.durationStr}
			ohlcv := OHLCVCrypto{DB: db, Config: config}

			if tt.shouldPanic {
				require.Panics(t, func() { _ = ohlcv.getModel() })
			} else {
				tx := ohlcv.getModel()
				require.Equal(t, db.Model(tt.expected).Statement.Table, tx.Statement.Table)
			}
		})
	}
}
