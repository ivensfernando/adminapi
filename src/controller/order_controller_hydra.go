package controller

import (
	"context"
	"fmt"
	"math"
	"strategyexecutor/src/connectors"
	"strategyexecutor/src/model"
	"strategyexecutor/src/repository"
	"time"

	logger "github.com/sirupsen/logrus"
)

// OrderControllerHydra executes the main trading flow based on the latest trading signal.
func OrderControllerHydra(
	ctx context.Context,
	c *connectors.GooeyClient,
	user *model.User,
	exchangeID uint,
	targetSymbol string, // BTCUSD
	targetExchange string, // hydra
) error {
	config := connectors.GetConfig()
	instrumentID := config.HydraInstrumentID
	hydraSymbol := config.HydraSymbol

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tradingSignalRepo := repository.NewTradingSignalRepository()
	exceptionRepo := repository.NewExceptionRepository()
	orderRepo := repository.NewOrderRepository()

	// ------------------------------------------------------------------
	// 1) Fetch the latest TradingSignal (from read-only DB)
	// ------------------------------------------------------------------
	signals, err := tradingSignalRepo.FindLatest(ctx, targetSymbol, targetExchange, 1)
	if err != nil {
		logger.WithError(err).Error("hydra - failed to fetch latest trading signal")
		Capture(
			ctx,
			exceptionRepo,
			"OrderController",
			"controller",
			"tradingSignalRepo.FindLatest",
			"error",
			err,
			map[string]interface{}{},
		)
		return err
	}
	if len(signals) == 0 {
		logger.Warn("hydra - no trading signals found")
		return nil
	}

	signal := signals[0]

	// ------------------------------------------------------------------
	// 2) Check if an order already exists for this signal
	// ------------------------------------------------------------------

	existingOrder, err := orderRepo.FindByExternalIDAndUserID(ctx, user.ID, signal.ID, model.OrderDirectionEntry)
	if err != nil {
		logger.WithError(err).Error("hydra - failed to fetch latest trading signal")
		Capture(
			ctx,
			exceptionRepo,
			"OrderController",
			"controller",
			"orderRepo.FindByExternalIDAndUser",
			"error",
			err,
			map[string]interface{}{},
		)
		logger.WithError(err).Error("hydra - failed to search for existing order")
		return err
	}

	if existingOrder != nil {
		logger.WithField("order_id", existingOrder.ID).
			Info("hydra - order already exists for this signal, checking status")

		if existingOrder.Status == model.OrderExecutionStatusFilled {
			logger.WithField("order_id", existingOrder.ID).
				Info("hydra - order already filled, skipping")
			return nil
		}

	}

	newOrder := &model.Order{
		UserID:     user.ID,
		ExchangeID: exchangeID, // hydra
		ExternalID: signal.ID,
		Symbol:     hydraSymbol,                      //signal.Symbol, "BTCUSDT"
		Side:       FirstLetterUpper(signal.Action),  // buy/sell
		PosSide:    FirstLetterUpper(signal.OrderID), //Short/Long
		OrderType:  "market",
		Quantity:   config.HydraQTD, //
		Status:     model.OrderExecutionStatusPending,
	}
	if err := orderRepo.CreateWithAutoLog(ctx, newOrder); err != nil {
		logger.WithError(err).Error("hydra - failed to create order with auto log")
		return err
	}

	// 2. Login
	if err := c.Login(ctx); err != nil {
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"hydra - Login failed",
		)
		return fmt.Errorf("hydra - Login failed: %v", err)
	}
	if c.SessionCookie == nil {
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"hydra - expected session cookie to be set after login",
		)
		return fmt.Errorf("hydra - expected session cookie to be set after login")
	}
	logger.Errorf("hydra - Got session cookie: %s", c.SessionCookie.Value)

	// 3. Fetch CSRF
	if err := c.FetchCSRF(ctx); err != nil {
		return fmt.Errorf("FetchCSRF failed: %v", err)
	}
	if c.CSRFTok == "" {
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"hydra - expected non empty CSRF token",
		)
		return fmt.Errorf("hydra - expected non empty CSRF token")
	}
	logger.Infof("CSRF: %s", c.CSRFTok)

	if c.DxtfidCookie == nil {
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"hydra - expected Dxtfid cookie to be set after login",
		)
		return fmt.Errorf("hydra - expected Dxtfid cookie to be set after login")
	}
	logger.Infof("hydra - Got Dxtfid cookie: %s", c.DxtfidCookie.Value)

	if err := c.InitAtmosphereTrackingID(ctx); err != nil {
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"hydra - init tracking id failed",
		)
		return fmt.Errorf(fmt.Sprintf("hydra - init tracking id failed: %v", err))
	}

	logger.Infof("hydra - AtmosphereTrackingID: %s", c.AtmosphereTrackingID)

	// 4. Close any open positions from the trade journal over the last 7 days
	start := time.Now().Add(-(time.Hour * 24 * 7))
	end := time.Now().UTC()

	time.Sleep(1 * time.Second)
	if err := c.CloseAllOpenFromTradeJournal(ctx, start, end); err != nil {
		//return fmt.Errorf("CloseAllOpenFromTradeJournal error: %v", err)
		logger.Warnf("hydra - CloseAllOpenFromTradeJournal error: %v", err)
	}

	if signal.Price == nil {
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"hydra - no trading signal price found",
		)
		logger.Error("hydra - no trading signal price found")
		return nil
	}

	orderSide := connectors.SideBuy
	qty := config.HydraQTD
	if signal.Action == "sell" {
		orderSide = connectors.SideSell
		qty = -qty
	}

	stoploss := connectors.CalcStopLoss(*signal.Price, config.HydraSLPercent, signal.Action)
	offset := math.Abs(*signal.Price - stoploss)

	resp, status, err := c.PlaceMarketOrder(
		ctx,
		instrumentID,
		hydraSymbol,
		qty,
		orderSide,
		connectors.PositionOpen,
		connectors.WithStopLoss(stoploss, offset, qty),
		connectors.WithRequestID(""),
	)
	if err != nil {
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"hydra - PlaceMarketOrder failed",
		)
		return fmt.Errorf("hydra - PlaceMarketOrder failed: %v", err)
	}

	logger.Infof("hydra - order status: %d", status)
	logger.Infof("hydra - order resp: %s", string(resp))
	if status < 200 || status >= 300 {
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"hydra - unexpected order status code",
		)
		return fmt.Errorf("hydra - unexpected order status code: %d, body=%s", status, string(resp))
	}

	if err := orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusFilled, "order placed on Hydra successfully"); err != nil {
		return fmt.Errorf("hydra - failed to  UpdateStatusWithAutoLog : %v", err)
	}

	logger.WithField("order_id", newOrder.ID).
		Info("hydra - order successfully completed")

	return nil
}
