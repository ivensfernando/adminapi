package ohlcvcrypto

import (
	"testing"
	"time"

	"github.com/nntaoli-project/goex"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Test fetching OHLCV data directly from Binance without mocks.
func TestFetchOHLCVFromBinance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
		return
	}
	db, _ := setupDBMock(t)

	// Set up the configuration
	config := &Config{
		Symbol:      "BTC",
		Quote:       "USDT",
		StartDt:     time.Now().Add(-24 * time.Hour),
		EndDt:       time.Now(),
		DurationStr: "1h",
		Limit:       1000,
	}

	targetSymbol := goex.NewCurrencyPair(goex.Currency{Symbol: config.Symbol}, goex.Currency{Symbol: config.Quote})

	// Initialize the OHLCVCrypto structure
	ohlcv := OHLCVCrypto{
		Log:    logrus.NewEntry(logrus.New()),
		DB:     db,
		Config: config,
	}

	// Set up the Binance API client
	ohlcv.exchange = ohlcv.newBinanceInstance()

	ticker, err := ohlcv.exchange.GetTicker(targetSymbol)
	require.NoError(t, err)
	require.Equal(t, "BTC_USDT", ticker.Pair.String())

	// Fetch the OHLCV series
	klines, err := ohlcv.fetchOHLCVSeries()

	require.NoError(t, err, "Should fetch OHLCV data without error")
	require.NotEmpty(t, klines, "Should return non-empty OHLCV data")

	// Optionally, log some data to verify correctness
	for _, k := range klines {
		t.Logf("Time: %v, Open: %v, High: %v, Low: %v, Close: %v, Volume: %v",
			time.Unix(k.Timestamp, 0).UTC(), k.Open, k.High, k.Low, k.Close, k.Vol)
	}
}
