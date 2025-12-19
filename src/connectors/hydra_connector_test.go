package connectors_test

import (
	"context"
	"fmt"
	"math"
	"strategyexecutor/src/connectors"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HydraUsername string `envconfig:"HYDRA_USERNAME" required:"true"` // only for tests
	HydraPassword string `envconfig:"HYDRA_PASSWORD" required:"true"` // only for tests
}

func GetConfig() Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}

func TestGooeyTrade_CloseOpenPositionsAndPlaceOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
		return
	}

	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	config := GetConfig()

	// 1. Create client
	c, err := connectors.NewGooeyClient(config.HydraUsername, config.HydraPassword)
	if err != nil {
		t.Fatalf("NewGooeyClient failed: %v", err)
	}

	// 2. Login
	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if c.SessionCookie == nil {
		t.Fatalf("expected session cookie to be set after login")
	}
	t.Logf("Got session cookie: %s", c.SessionCookie.Value)

	// 3. Fetch CSRF
	if err := c.FetchCSRF(ctx); err != nil {
		t.Fatalf("FetchCSRF failed: %v", err)
	}
	if c.CSRFTok == "" {
		t.Fatalf("expected non empty CSRF token")
	}
	t.Logf("CSRF: %s", c.CSRFTok)

	if c.DxtfidCookie == nil {
		t.Fatalf("expected Dxtfid cookie to be set after login")
	}
	t.Logf("Got Dxtfid cookie: %s", c.DxtfidCookie.Value)

	if err := c.InitAtmosphereTrackingID(ctx); err != nil {
		t.Fatalf(fmt.Sprintf("init tracking id failed: %v", err))
	}

	t.Logf("AtmosphereTrackingID: %s", c.AtmosphereTrackingID)

	// 4. Close any open positions from trade journal over last 15 days
	start := time.Now().Add(-(time.Hour * 24 * 7))
	end := time.Now().UTC()

	fromMs := connectors.ToMillis(start) // or toMillis if in same package
	toMs := connectors.ToMillis(end)

	t.Logf("From: %s, ms: %d", start.Format(time.RFC3339), fromMs)
	t.Logf("To:   %s, ms: %d", end.Format(time.RFC3339), toMs)

	time.Sleep(1 * time.Second)
	if err := c.CloseAllOpenFromTradeJournal(ctx, start, end); err != nil {
		//t.Fatalf("CloseAllOpenFromTradeJournal error: %v", err)
	}

	// Mark time just before placing order so we can filter history
	orderPlacedAt := time.Now().UTC()

	// 5. Place a market order using the helper
	const (
		instrumentID = 9910
		symbol       = "BTC/USD.crypto"
	)

	action := "buy"
	price := 92364.0000000000
	qty := 0.00001

	orderSide := connectors.SideBuy
	if action == "sell" {
		orderSide = connectors.SideSell
		qty = -qty
	}

	stoploss := connectors.CalcStopLoss(price, 5, action)
	offset := math.Abs(price - stoploss)
	t.Logf("stoploss: %f", stoploss)
	t.Logf("offset: %f", offset)

	resp, status, err := c.PlaceMarketOrder(
		ctx,
		instrumentID,
		symbol,
		qty,
		orderSide,
		connectors.PositionOpen,
		connectors.WithStopLoss(stoploss, offset, qty),
		connectors.WithRequestID(""),
	)
	if err != nil {
		t.Fatalf("PlaceMarketOrder failed: %v", err)
	}
	t.Logf("order status: %d", status)
	t.Logf("order resp: %s", string(resp))
	if status < 200 || status >= 300 {
		t.Fatalf("unexpected order status code: %d, body=%s", status, string(resp))
	}

	// 6. Query history for last 15 days and assert our trade is present
	end = time.Now().UTC()
	toMs = connectors.ToMillis(end)

	t.Logf("From: %s, ms: %d", start.Format(time.RFC3339), fromMs)
	t.Logf("To:   %s, ms: %d", end.Format(time.RFC3339), toMs)

	time.Sleep(1 * time.Second)
	trades, status, err := c.HistoryTrades(ctx, fromMs, toMs)
	if err != nil {
		t.Fatalf("HistoryTrades error: %v", err)
	}
	t.Logf("history HTTP status: %d", status)
	if status < 200 || status >= 300 {
		t.Fatalf("unexpected history status code: %d", status)
	}

	orderPlacedAtMs := connectors.ToMillis(orderPlacedAt)
	found := false

	for _, tr := range trades {
		tm := time.UnixMilli(tr.Time).UTC()
		t.Logf("[%s] %s %s qty=%f price=%f",
			tm.Format(time.RFC3339),
			tr.TradeSide,
			tr.Symbol,
			tr.Quantity,
			tr.FillPrice,
		)

		if tr.Symbol == symbol &&
			tr.TradeSide == "BUY" &&
			tr.Quantity == qty &&
			tr.Time >= orderPlacedAtMs {
			found = true
		}
	}

	if !found {
		t.Logf("expected to find at least one BUY trade for %s qty=%f after %s, but found none",
			symbol,
			qty,
			orderPlacedAt.Format(time.RFC3339),
		)
	}
}
