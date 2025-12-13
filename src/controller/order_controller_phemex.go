package controller

import (
	"adminapi/src/mapper"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	logger "github.com/sirupsen/logrus"

	"adminapi/src/connectors"
	"adminapi/src/model"
	"adminapi/src/repository"
)

func FirstLetterUpper(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// OrderController executes the main trading flow based on the latest trading signal.
func OrderController(
	ctx context.Context,
	phemexClient *connectors.Client,
	user string,
	orderSizePercent int,
	exchangeID uint,
) error {

	logger.Debugf("OrderController INITIALIZED ")
	logger.Info("starting order controller flow")

	tradingSignalRepo := repository.NewTradingSignalRepository()
	phemexRepo := repository.NewPhemexOrderRepository()
	exceptionRepo := repository.NewExceptionRepository()
	orderRepo := repository.NewOrderRepository()

	// ------------------------------------------------------------------
	// 1) Fetch the latest TradingSignal (from read-only DB)
	// ------------------------------------------------------------------
	signals, err := tradingSignalRepo.FindLatest(ctx, 1)
	if err != nil {
		logger.WithError(err).Error("failed to fetch latest trading signal")
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
		logger.Warn("no trading signals found")
		return nil
	}

	signal := signals[0]
	symbol := NormalizeToUSDT(signal.Symbol)
	logger.WithFields(map[string]interface{}{
		"user":          user,
		"signal_id":     signal.ID,
		"signal.Symbol": signal.Symbol,
		"symbol":        symbol,
		"action":        signal.Action,
	}).Info("latest trading signal fetched")

	// ------------------------------------------------------------------
	// 2) Check if an order already exists for this signal
	// ------------------------------------------------------------------

	existingOrder, err := orderRepo.FindByExternalIDAndUser(ctx, user, signal.ID)
	if err != nil {
		logger.WithError(err).Error("failed to fetch latest trading signal")
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
		logger.WithError(err).Error("failed to search for existing order")
		return err
	}

	if existingOrder != nil {
		logger.WithField("order_id", existingOrder.ID).
			Info("order already exists for this signal, checking status")

		if existingOrder.Status == model.OrderExecutionStatusFilled {
			logger.WithField("order_id", existingOrder.ID).
				Info("order already filled, skipping")
			return nil
		}

	}

	baseSymbol, baseAvail, usdtAvail, price, err := phemexClient.GetAvailableBaseFromUSDT(symbol)
	logger.WithField("baseSymbol", baseSymbol).
		WithField("baseAvail", baseAvail).
		WithField("usdtAvail", usdtAvail).
		WithField("price", price).
		WithField("OrderSizePercent", orderSizePercent).
		Debug("GetAvailableBaseFromUSDT")

	value := PercentOfFloatSafe(baseAvail, orderSizePercent)

	logger.WithField("value", value).
		WithField("Symbol", symbol).
		Debug("Value of order in ")
	// ------------------------------------------------------------------
	// 3) Create new Order (Phemex = exchange_id 1)
	// ------------------------------------------------------------------

	newOrder := &model.Order{
		UserID:     user,
		ExchangeID: exchangeID, // Phemex
		ExternalID: signal.ID,
		Symbol:     symbol,                           //signal.Symbol, "BTCUSDT"
		Side:       FirstLetterUpper(signal.Action),  // buy/sell
		PosSide:    FirstLetterUpper(signal.OrderID), //Short/Long
		OrderType:  "market",
		Quantity:   value, //
		Status:     model.OrderExecutionStatusFilled,
	}

	if err := orderRepo.CreateWithAutoLog(ctx, newOrder); err != nil {
		logger.WithError(err).Error("failed to create order with auto log")
		return err
	}

	logger.WithField("order_id", newOrder.ID).Info("new order created")

	// ------------------------------------------------------------------
	// 4) Close all existing positions for this symbol on Phemex
	// ------------------------------------------------------------------

	if err := phemexClient.CloseAllPositions(newOrder.Symbol); err != nil {
		logger.WithError(err).
			WithField("symbol", newOrder.Symbol).
			Error("failed to close all positions")

		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to close existing positions",
		)

		return err
	}

	logger.WithField("symbol", newOrder.Symbol).
		Info("all previous positions closed")

	// ------------------------------------------------------------------
	// 5) Place new Market Order on Phemex
	// ------------------------------------------------------------------
	quantityStr := strconv.FormatFloat(newOrder.Quantity, 'f', 4, 64)

	resp, err := phemexClient.PlaceOrder(
		newOrder.Symbol,
		newOrder.Side,
		newOrder.PosSide,
		quantityStr,
		"Market",
		false,
	)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"symbol":  newOrder.Symbol,
			"side":    newOrder.Side,
			"posSide": newOrder.PosSide,
			"qty":     quantityStr,
		}).WithError(err).Error("failed to place order on Phemex")

		Capture(
			ctx,
			exceptionRepo,
			"OrderController",
			"controller",
			"phemexClient.PlaceOrder",
			"error",
			err,
			map[string]interface{}{
				"symbol": newOrder.Symbol,
				"side":   newOrder.Side,
				"qty":    quantityStr,
			},
		)
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to place order on Phemex",
		)

		return err // ou continue, dependendo do fluxo
	}

	if resp.Code != 0 {
		logger.WithFields(map[string]interface{}{
			"symbol": newOrder.Symbol,
			"code":   resp.Code,
			"msg":    resp.Msg,
		}).Error("Phemex returned non-zero code")

		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"phemex returned non-zero code while placing order",
		)

		return fmt.Errorf("phemex error %d: %s", resp.Code, resp.Msg)
	}

	var payload model.PhemexOrderResponse

	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		logger.WithFields(map[string]interface{}{
			"symbol": newOrder.Symbol,
		}).WithError(err).Error("failed to unmarshal phemex response payload")

		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to decode phemex response",
		)

		return err
	}

	// Map API payload -> DB model (versão safe)
	ord, err := mapper.MapPhemexResponseToModel(&payload, newOrder.ID)
	if err != nil {
		logger.WithError(err).Error("failed to map phemex response to model")

		Capture(
			ctx,
			exceptionRepo,
			"OrderController",
			"controller",
			"mapper.MapPhemexResponseToModel",
			"error",
			err,
			map[string]interface{}{},
		)
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to map phemex response to model",
		)

		return err
	}

	// Persist Phemex order in DB
	if err := phemexRepo.Create(ctx, ord); err != nil {
		logger.WithError(err).Error("failed to persist phemex order")

		Capture(
			ctx,
			exceptionRepo,
			"OrderController",
			"controller",
			"phemexRepo.Create",
			"error",
			err,
			map[string]interface{}{
				"symbol": newOrder.Symbol,
				"side":   newOrder.Side,
				"qty":    quantityStr,
			},
		)
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to persist phemex order",
		)

		return err
	} else {
		if err := orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusPending, "order placed on Phemex successfully"); err != nil {
		}
	}

	// opcional: salvar JSON bruto da resposta na tabela Order
	rawJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		logger.WithError(err).Error("failed to marshal phemex raw response for storage")
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to persist phemex order",
		)
		Capture(
			ctx,
			exceptionRepo,
			"OrderController",
			"controller",
			"json.MarshalIndent(resp.Data, \"\", \"  \")",
			"error",
			err,
			map[string]interface{}{},
		)
	} else {
		if err := orderRepo.UpdateResp(ctx, newOrder.ID, string(rawJSON), model.OrderExecutionStatusPending); err != nil {
			logger.WithError(err).Error("failed to update order exchange_resp")
		}
	}

	pos, err := phemexClient.GetPositionsUSDT()
	if err != nil {
		logger.WithError(err).Error("failed to get positions on Phemex")
		Capture(
			ctx,
			exceptionRepo,
			"OrderController",
			"controller",
			"phemexClient.GetPositionsUSDT",
			"error",
			err,
			map[string]interface{}{},
		)
	}
	logger.WithField("positions", pos).Info("positions on Phemex")

	logger.WithFields(map[string]interface{}{
		"order_id": newOrder.ID,
		//"exchange_order": apiResp.OrderID,
	}).Info("order placed on Phemex successfully")

	for _, p := range pos.Positions {
		if p.SizeRq == "" || p.SizeRq == "0" {
			continue
		}
		if p.Symbol == newOrder.Symbol {
			// ------------------------------------------------------------------
			// 6) Update the Order as Executed / Filled
			// ------------------------------------------------------------------
			if err := orderRepo.UpdateStatusWithAutoLog(
				ctx,
				newOrder.ID,
				model.OrderExecutionStatusFilled,
				"order executed successfully on phemex",
			); err != nil {
				logger.WithError(err).Error("failed to update order final status")
				Capture(
					ctx,
					exceptionRepo,
					"OrderController",
					"controller",
					"orderRepo.UpdateStatusWithAutoLog",
					"error",
					err,
					map[string]interface{}{
						"symbol": newOrder.Symbol,
						"side":   newOrder.Side,
						"qty":    quantityStr,
					},
				)
				return err
			}

			logger.WithField("order_id", newOrder.ID).
				Info("order successfully completed")
		}

	}

	return nil
}
