package connectors_test

import (
	"context"
	"fmt"
	"math"
	"strategyexecutor/src/connectors"
	"strings"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
	logger "github.com/sirupsen/logrus"
)

type ConfigKraken struct {
	KrakenAPIKEY    string `envconfig:"KRAKEN_API_KEY" required:"true"`    // only for tests
	KrakenAPISECRET string `envconfig:"KRAKEN_API_SECRET" required:"true"` // only for tests
}

func GetConfigKraken() ConfigKraken {
	var config ConfigKraken
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}

func TestKrakenFutures_BasicFlow_MarketOrder_StopLoss_Verify(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
		return
	}

	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cfg := GetConfigKraken()
	c := connectors.NewKrakenFuturesClient(cfg.KrakenAPIKEY, cfg.KrakenAPISECRET, "")

	const symbol = "PF_XBTUSD"

	// Best-effort cleanup at the end (uncomment once you are confident).
	defer func() {
		_, _ = c.CancelAllOrders(symbol)
		_ = c.CloseAllPositions(symbol)
	}()

	t.Run("pre-clean: cancel orders and close positions", func(t *testing.T) {
		if _, err := c.CancelAllOrders(symbol); err != nil {
			t.Fatalf("CancelAllOrders failed: %v", err)
		}
		if err := c.CloseAllPositions(symbol); err != nil {
			t.Fatalf("CloseAllPositions failed: %v", err)
		}

		waitUntil(t, ctx, 20*time.Second, 500*time.Millisecond, func() (bool, string) {
			pos := mustGetOpenPositions(t, c)
			p := findPosition(pos, symbol)
			if p == nil || p.Size == 0 {
				return true, "no open position"
			}
			return false, fmt.Sprintf("still open position: side=%s size=%f", p.Side, p.Size)
		})
	})

	var (
		placedAt        time.Time
		entryCliOrdID   string
		entryOrderID    string
		entrySide       = "sell" // requested example, but keep logic correct
		expectedPosSide = "short"
		entrySize       = 0.0001
	)

	t.Run("place market order", func(t *testing.T) {
		placedAt = time.Now().UTC()
		entryCliOrdID = fmt.Sprintf("go-%d", time.Now().UnixNano())

		reduceOnly := false
		resp, err := c.SendOrder(connectors.SendOrderRequest{
			OrderType:  "mkt",
			Symbol:     symbol,
			Side:       entrySide,
			Size:       entrySize,
			ReduceOnly: &reduceOnly,
			CliOrdID:   &entryCliOrdID,
		})
		if err != nil {
			t.Fatalf("SendOrder failed: %v", err)
		}
		if resp == nil {
			t.Fatalf("SendOrder returned nil response")
		}
		if resp.Result != "success" {
			t.Fatalf("SendOrder expected result=success, got %q", resp.Result)
		}
		if resp.SendStatus.OrderID == "" {
			t.Fatalf("SendOrder expected non-empty sendStatus.order_id")
		}

		switch resp.SendStatus.Status {
		case "placed", "partiallyFilled", "filled":
		default:
			t.Fatalf("SendOrder unexpected sendStatus.status=%q (order_id=%s)", resp.SendStatus.Status, resp.SendStatus.OrderID)
		}

		entryOrderID = resp.SendStatus.OrderID

		t.Logf("placed market order: cliOrdId=%s order_id=%s status=%s placedAt=%s side=%s size=%f",
			entryCliOrdID, entryOrderID, resp.SendStatus.Status, placedAt.Format(time.RFC3339), entrySide, entrySize)
	})

	var openedPos *connectors.OpenPosition

	t.Run("verify open position exists and direction matches", func(t *testing.T) {
		waitUntil(t, ctx, 20*time.Second, 500*time.Millisecond, func() (bool, string) {
			pos := mustGetOpenPositions(t, c)
			p := findPosition(pos, symbol)
			if p == nil {
				return false, "position not found yet"
			}
			if strings.ToLower(p.Side) != expectedPosSide {
				return false, fmt.Sprintf("unexpected position side=%s expected=%s size=%f", p.Side, expectedPosSide, p.Size)
			}
			if p.Size <= 0 {
				return false, fmt.Sprintf("size not positive yet. side=%s size=%f", p.Side, p.Size)
			}
			openedPos = p
			return true, fmt.Sprintf("ok: side=%s size=%f price=%v fillTime=%s", p.Side, p.Size, p.Price, p.FillTime)
		})

		final := mustGetOpenPositions(t, c)
		if p := findPosition(final, symbol); p != nil {
			t.Logf("final position: symbol=%s side=%s size=%f price=%v fillTime=%s (placedAt=%s)",
				p.Symbol, p.Side, p.Size, p.Price, p.FillTime, placedAt.Format(time.RFC3339))
		} else {
			t.Fatalf("position disappeared unexpectedly after verification")
		}
	})

	var (
		slCliOrdID   string
		slOrderID    string
		stopPrice    float64
		stopSide     string
		stopQuantity float64
	)

	t.Run("place stop loss reduceOnly (stp)", func(t *testing.T) {
		if openedPos == nil {
			t.Fatalf("openedPos is nil, cannot place stop loss")
		}

		entryPrice := derefFloat64(openedPos.Price)
		if entryPrice <= 0 {
			t.Fatalf("invalid entry price from openpositions: %v", openedPos.Price)
		}

		// Choose a percent for the test. Keep it small to avoid silly values.
		// You can wire this from config if you want.
		const slPct = 0.30

		// CalcStopLoss expects action = "buy" or "sell" (your existing helper).
		stopPrice = math.Round(connectors.CalcStopLoss(entryPrice, slPct, entrySide))

		logger.Infof("stopPrice: %f", stopPrice)

		// Stop side must be opposite to close the position.
		// If we are short (sell), stop closes with buy.
		stopSide = "buy"
		if entrySide == "buy" {
			stopSide = "sell"
		}

		// Use the full open position size. Kraken expects positive size.
		stopQuantity = math.Abs(openedPos.Size)
		if stopQuantity <= 0 {
			t.Fatalf("invalid opened position size: %f", openedPos.Size)
		}

		stopReduceOnly := true
		slCliOrdID = fmt.Sprintf("go-sl-%d", time.Now().UnixNano())

		resp, err := c.SendOrder(connectors.SendOrderRequest{
			OrderType:  "stp",
			Symbol:     symbol,
			Side:       stopSide,
			Size:       stopQuantity,
			StopPrice:  &stopPrice,
			ReduceOnly: &stopReduceOnly,
			CliOrdID:   &slCliOrdID,
		})
		if err != nil {
			t.Fatalf("SendOrder(stop) failed: %v", err)
		}
		if resp == nil {
			t.Fatalf("SendOrder(stop) returned nil response")
		}
		if resp.Result != "success" {
			t.Fatalf("SendOrder(stop) expected result=success, got %q", resp.Result)
		}
		if resp.SendStatus.OrderID == "" {
			t.Fatalf("SendOrder(stop) expected non-empty sendStatus.order_id")
		}
		slOrderID = resp.SendStatus.OrderID

		switch resp.SendStatus.Status {
		case "placed", "partiallyFilled", "filled":
		default:
			t.Fatalf("SendOrder(stop) unexpected sendStatus.status=%q (order_id=%s)", resp.SendStatus.Status, resp.SendStatus.OrderID)
		}

		t.Logf("placed stop loss: cliOrdId=%s order_id=%s status=%s stopSide=%s stopQty=%f stopPrice=%f entryPrice=%f",
			slCliOrdID, slOrderID, resp.SendStatus.Status, stopSide, stopQuantity, stopPrice, entryPrice)
	})

	t.Run("verify stop loss exists in open orders", func(t *testing.T) {
		// This requires your Kraken client to implement GetOpenOrdersRaw (as previously coded).
		raw, err := c.GetOpenOrdersRaw()
		if err != nil {
			t.Fatalf("GetOpenOrdersRaw failed: %v", err)
		}
		body := string(raw)

		// Verify either client order id or order id is present.
		if !strings.Contains(body, slCliOrdID) && !strings.Contains(body, slOrderID) {
			t.Fatalf("expected stop order to be present in open orders. missing cliOrdId=%s and order_id=%s. body=%s",
				slCliOrdID, slOrderID, body)
		}
		t.Logf("verified stop order present in open orders (cliOrdId=%s order_id=%s)", slCliOrdID, slOrderID)
	})
}

func mustGetOpenPositions(t *testing.T, c *connectors.KrakenFuturesClient) *connectors.OpenPositionsResponse {
	t.Helper()
	resp, err := c.GetOpenPositions()
	if err != nil {
		t.Fatalf("GetOpenPositions failed: %v", err)
	}
	if resp == nil {
		t.Fatalf("GetOpenPositions returned nil response")
	}
	if resp.Result != "success" {
		t.Fatalf("GetOpenPositions expected result=success, got %q", resp.Result)
	}
	return resp
}

func findPosition(resp *connectors.OpenPositionsResponse, symbol string) *connectors.OpenPosition {
	if resp == nil {
		return nil
	}
	for i := range resp.OpenPositions {
		if resp.OpenPositions[i].Symbol == symbol {
			return &resp.OpenPositions[i]
		}
	}
	return nil
}

func derefFloat64(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func waitUntil(t *testing.T, ctx context.Context, max time.Duration, step time.Duration, cond func() (bool, string)) {
	t.Helper()

	deadline := time.Now().Add(max)
	var last string

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			t.Fatalf("context done while waiting: %v. last=%s", ctx.Err(), last)
		default:
		}

		ok, msg := cond()
		last = msg
		if ok {
			t.Logf("waitUntil satisfied: %s", msg)
			return
		}
		time.Sleep(step)
	}

	t.Fatalf("waitUntil timeout after %s. last=%s", max.String(), last)
}
