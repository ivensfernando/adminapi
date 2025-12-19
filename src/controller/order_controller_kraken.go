package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strategyexecutor/src/connectors"
	"strategyexecutor/src/model"
	"strategyexecutor/src/repository"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
)

// OrderControllerKrakenFutures executes the main trading flow based on the latest trading signal.
// Flow:
// 1) fetch latest signal
// 2) skip if already filled order exists for this signal
// 3) cancel all orders for symbol
// 4) if an opposite position exists, close it
// 5) place market order in signal direction
// 6) verify by openpositions that position exists and matches direction
// 7) place reduceOnly stop-loss (stp) for the full open position size
func OrderControllerKrakenFutures(
	ctx context.Context,
	c *connectors.KrakenFuturesClient,
	user *model.User,
	exchangeID uint,
	targetSymbol string, // BTCUSD
	targetExchange string, // kraken
) error {
	config := connectors.GetConfig()
	krakenSymbol := config.KrakenSymbol

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tradingSignalRepo := repository.NewTradingSignalRepository()
	exceptionRepo := repository.NewExceptionRepository()
	orderRepo := repository.NewOrderRepository()

	// ------------------------------------------------------------------
	// 1) Fetch latest TradingSignal
	// ------------------------------------------------------------------
	signals, err := tradingSignalRepo.FindLatest(ctx, targetSymbol, targetExchange, 1)
	if err != nil {
		logger.WithError(err).Error("kraken - failed to fetch latest trading signal")
		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKrakenFutures",
			"controller",
			"tradingSignalRepo.FindLatest",
			"error",
			err,
			map[string]interface{}{},
		)
		return err
	}
	if len(signals) == 0 {
		logger.Warn("kraken - no trading signals found")
		return nil
	}
	signal := signals[0]

	// ------------------------------------------------------------------
	// 2) Check if we already processed this signal
	// ------------------------------------------------------------------
	existingOrder, err := orderRepo.FindByExternalIDAndUserID(ctx, user.ID, signal.ID, model.OrderDirectionEntry)
	if err != nil {
		logger.WithError(err).Error("kraken - failed to search for existing order")
		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKrakenFutures",
			"controller",
			"orderRepo.FindByExternalIDAndUser",
			"error",
			err,
			map[string]interface{}{},
		)
		return err
	}
	if existingOrder != nil {
		logger.WithField("order_id", existingOrder.ID).Info("kraken - order already exists for this signal, checking status")
		if existingOrder.Status == model.OrderExecutionStatusFilled {
			logger.WithField("order_id", existingOrder.ID).Info("kraken - order already filled, skipping")
			return nil
		}
	}

	// ------------------------------------------------------------------
	// 3) Persist local order row early
	// ------------------------------------------------------------------
	desiredSide := normalizeKrakenSide(signal.Action) // buy/sell
	desiredPosSide := desiredPositionSide(desiredSide)

	newOrder := &model.Order{
		UserID:     user.ID,
		ExchangeID: exchangeID, // kraken futures
		ExternalID: signal.ID,
		Symbol:     krakenSymbol,
		Side:       FirstLetterUpper(desiredSide),    // Buy/Sell
		PosSide:    FirstLetterUpper(desiredPosSide), // Long/Short
		OrderType:  "market",
		Quantity:   config.KrakenQTD, // add to config
		Status:     model.OrderExecutionStatusPending,
	}
	if err := orderRepo.CreateWithAutoLog(ctx, newOrder); err != nil {
		logger.WithError(err).Error("kraken - failed to create order with auto log")
		return err
	}

	fail := func(msg string, e error) error {
		_ = orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusError, msg)
		if e != nil {
			return fmt.Errorf("%s: %w", msg, e)
		}
		return fmt.Errorf("%s", msg)
	}

	// 4) Pre-clean: cancel orders, close positions, verify flat
	if _, err := c.CancelAllOrders(krakenSymbol); err != nil {
		return fail("CancelAllOrders failed", err)
	}

	if err := c.CloseAllPositions(krakenSymbol); err != nil {
		return fail("CloseAllPositions failed", err)
	}

	if err := waitUntil(ctx, 15*time.Second, 500*time.Millisecond, func() (bool, string, error) {
		pos, err := c.GetOpenPositions()
		if err != nil {
			return false, "GetOpenPositions failed", err
		}
		p := findKrakenPosition(pos, krakenSymbol)
		if p == nil || p.Size == 0 {
			return true, "no open position", nil
		}
		return false, fmt.Sprintf("still open position: side=%s size=%f", p.Side, p.Size), nil
	}); err != nil {
		return fail("expected no open position after CloseAllPositions", err)
	}

	// ------------------------------------------------------------------
	// 6) Place market order
	// ------------------------------------------------------------------
	cliOrdID := fmt.Sprintf("go-%d", time.Now().UnixNano())
	reduceOnly := false

	sendResp, err := c.SendOrder(connectors.SendOrderRequest{
		OrderType:  "mkt",
		Symbol:     krakenSymbol,
		Side:       desiredSide,
		Size:       config.KrakenQTD,
		ReduceOnly: &reduceOnly,
		CliOrdID:   &cliOrdID,
	})
	if err != nil {
		return fail("kraken - SendOrder (market) failed", err)
	}
	if sendResp == nil || sendResp.Result != "success" {
		return fail("kraken - SendOrder (market) returned non-success result", nil)
	}

	logger.WithFields(map[string]interface{}{
		"symbol":     krakenSymbol,
		"side":       desiredSide,
		"size":       config.KrakenQTD,
		"cliOrdId":   cliOrdID,
		"order_id":   sendResp.SendStatus.OrderID,
		"status":     sendResp.SendStatus.Status,
		"serverTime": sendResp.ServerTime,
	}).Info("kraken - market order sent")

	// ------------------------------------------------------------------
	// 7) Verify by openpositions that we have a position in the desired direction
	// ------------------------------------------------------------------
	var openedPos *connectors.OpenPosition
	verifyDeadline := time.Now().Add(15 * time.Second)

	for time.Now().Before(verifyDeadline) {
		select {
		case <-ctx.Done():
			return fail("kraken - context done while verifying open position", ctx.Err())
		default:
		}

		p, err := c.GetOpenPositions()
		if err != nil {
			return fail("kraken - GetOpenPositions failed during verification", err)
		}
		pos := findKrakenPosition(p, krakenSymbol)
		if pos != nil && pos.Size > 0 && pos.Side == desiredPosSide {
			openedPos = pos
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if openedPos == nil {
		return fail("kraken - market order verification failed (no matching open position found)", nil)
	}

	// ------------------------------------------------------------------
	// 8) Place stop-loss as reduceOnly stop order for the full open position size
	// ------------------------------------------------------------------
	// Prefer signal.Price if present. Otherwise use avg entry price from openpositions.
	entryPrice := derefFloat64(openedPos.Price)
	if signal.Price != nil && *signal.Price > 0 {
		entryPrice = *signal.Price
	}
	if entryPrice <= 0 {
		return fail("kraken - cannot compute stop loss, entry price is invalid", nil)
	}

	stopPrice := math.Round(connectors.CalcStopLoss(entryPrice, config.KrakenSLPercent, desiredSide))

	stopSide := oppositeOrderSide(desiredSide) // to close long: sell. to close short: buy
	stopReduceOnly := true
	stopCliOrdID := fmt.Sprintf("go-sl-%d", time.Now().UnixNano())

	// For Kraken: orderType=stp requires stopPrice. If no limitPrice is provided it triggers a market order.
	// We set reduceOnly so it can only reduce and never open a new position.
	slResp, err := c.SendOrder(connectors.SendOrderRequest{
		OrderType:  "stp",
		Symbol:     krakenSymbol,
		Side:       stopSide,
		Size:       openedPos.Size,
		StopPrice:  &stopPrice,
		ReduceOnly: &stopReduceOnly,
		CliOrdID:   &stopCliOrdID,
		// TriggerSignal can be set if you want. Defaults are exchange-side behavior.
		// TriggerSignal: ptrString("mark"),
	})
	if err != nil {
		return fail("kraken - SendOrder (stop loss) failed", err)
	}
	if slResp == nil || slResp.Result != "success" {
		return fail("kraken - SendOrder (stop loss) returned non-success result", nil)
	}

	logger.WithFields(map[string]interface{}{
		"symbol":      krakenSymbol,
		"pos_side":    openedPos.Side,
		"pos_size":    openedPos.Size,
		"entry_price": entryPrice,
		"sl_price":    stopPrice,
		"sl_side":     stopSide,
		"cliOrdId":    stopCliOrdID,
		"order_id":    slResp.SendStatus.OrderID,
		"status":      slResp.SendStatus.Status,
	}).Info("kraken - stop loss order sent")

	// Optional: if your client has GetOpenOrdersRaw, verify the stop order is present.
	type openOrdersGetter interface {
		GetOpenOrdersRaw() (json.RawMessage, error)
	}
	if oo, ok := any(c).(openOrdersGetter); ok {
		raw, err := oo.GetOpenOrdersRaw()
		if err != nil {
			logger.WithError(err).Warn("kraken - GetOpenOrdersRaw failed, skipping stop order presence check")
		} else if !jsonContains(raw, stopCliOrdID) && !jsonContains(raw, slResp.SendStatus.OrderID) {
			logger.WithFields(map[string]interface{}{
				"cliOrdId": stopCliOrdID,
				"order_id": slResp.SendStatus.OrderID,
			}).Warn("kraken - stop order not found in open orders response (non-fatal)")
		} else {
			logger.WithFields(map[string]interface{}{
				"cliOrdId": stopCliOrdID,
				"order_id": slResp.SendStatus.OrderID,
			}).Info("kraken - verified stop order is present in open orders")
		}
	}

	// ------------------------------------------------------------------
	// 9) Mark local order as filled
	// ------------------------------------------------------------------
	if err := orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusFilled, "order placed on Kraken Futures successfully (market + stop)"); err != nil {
		return fmt.Errorf("kraken - failed to UpdateStatusWithAutoLog: %w", err)
	}

	logger.WithField("order_id", newOrder.ID).Info("kraken - order successfully completed")
	return nil
}

func normalizeKrakenSide(action string) string {
	a := strings.ToLower(strings.TrimSpace(action))
	if a == "sell" {
		return "sell"
	}
	return "buy"
}

func desiredPositionSide(orderSide string) string {
	if orderSide == "sell" {
		return "short"
	}
	return "long"
}

func oppositeOrderSide(orderSide string) string {
	if orderSide == "sell" {
		return "buy"
	}
	return "sell"
}

func findKrakenPosition(resp *connectors.OpenPositionsResponse, symbol string) *connectors.OpenPosition {
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

func ptrFloat64ToString(v *float64) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%f", *v)
}

func derefFloat64(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func jsonContains(raw json.RawMessage, needle string) bool {
	if len(raw) == 0 || needle == "" {
		return false
	}
	return strings.Contains(string(raw), needle)
}

func waitUntil(
	ctx context.Context,
	max time.Duration,
	step time.Duration,
	cond func() (ok bool, msg string, err error),
) error {
	deadline := time.Now().Add(max)
	var last string

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context done while waiting: %w. last=%s", ctx.Err(), last)
		default:
		}

		ok, msg, err := cond()
		last = msg
		if err != nil {
			return fmt.Errorf("%s: %w", msg, err)
		}
		if ok {
			return nil
		}
		time.Sleep(step)
	}

	return fmt.Errorf("timeout after %s. last=%s", max.String(), last)
}
