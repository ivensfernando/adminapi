package controller

import (
	"adminapi/src/connectors"
	"adminapi/src/mapper"
	"adminapi/src/model"
	"adminapi/src/repository"
	"context"
	"encoding/json"
	"math"
	"strings"

	logger "github.com/sirupsen/logrus"
)

// OrderControllerKucoin executes the trading flow for KuCoin using the latest signal.
func OrderControllerKucoin(
	ctx context.Context,
	kucoinClient *connectors.KucoinConnector,
	user *model.User,
	orderSizePercent int,
	exchangeID uint,
	targetSymbol string, // BTCUSD
	targetExchange string,
) error {

	logger.Debugf("OrderControllerKucoin INITIALIZED ")
	logger.Info("starting kucoin order controller flow")

	tradingSignalRepo := repository.NewTradingSignalRepository()
	kucoinRepo := repository.NewKucoinOrderRepository()
	exceptionRepo := repository.NewExceptionRepository()
	orderRepo := repository.NewOrderRepository()

	signals, err := tradingSignalRepo.FindLatest(ctx, targetSymbol, targetExchange, 1)
	if err != nil {
		logger.WithError(err).Error("failed to fetch latest trading signal")
		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKucoin",
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
	kucoinSymbol := NormalizeToKucoinFuturesSymbol(symbol)
	logger.WithFields(map[string]interface{}{
		"user":          user.Username,
		"signal_id":     signal.ID,
		"signal.Symbol": signal.Symbol,
		"symbol":        kucoinSymbol,
		"action":        signal.Action,
	}).Info("latest trading signal fetched")

	existingOrder, err := orderRepo.FindByExternalIDAndUserID(ctx, user.ID, signal.ID)
	if err != nil {
		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKucoin",
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

	baseSymbol, baseAvail, usdtAvail, price, err := kucoinClient.GetAvailableBaseFromUSDT(kucoinSymbol)
	logger.WithField("baseSymbol", baseSymbol).
		WithField("baseAvail", baseAvail).
		WithField("usdtAvail", usdtAvail).
		WithField("price", price).
		WithField("OrderSizePercent", orderSizePercent).
		Debug("KuCoin GetAvailableBaseFromUSDT")

	if err != nil {
		logger.WithError(err).Error("failed to fetch available balance on KuCoin")
		Capture(ctx, exceptionRepo, "OrderControllerKucoin", "controller", "kucoinClient.GetAvailableBaseFromUSDT", "error", err, map[string]interface{}{})
		return err
	}

	contract, err := kucoinClient.GetFuturesContractInfo(kucoinSymbol)
	if err != nil {
		logger.WithError(err).Error("failed to fetch kucoin futures contract info")
		return err
	}

	usdToUse := PercentOfFloatSafe(usdtAvail, orderSizePercent)
	if usdToUse <= 0 {
		logger.Warn("calculated order size is zero, skipping order placement")
		return nil
	}

	baseToUse := usdToUse / price
	contractsFloat := baseToUse / contract.Multiplier
	lotStep := contract.LotSize
	if lotStep <= 0 {
		lotStep = 1
	}

	contracts := math.Floor(contractsFloat/lotStep) * lotStep
	if contracts <= 0 {
		logger.Warn("calculated contract size is below minimum lot size, skipping order placement")
		return nil
	}

	quantity := float64(int64(contracts))

	newOrder := &model.Order{
		UserID:     user.ID,
		ExchangeID: exchangeID,
		ExternalID: signal.ID,
		Symbol:     kucoinSymbol,
		Side:       FirstLetterUpper(signal.Action),
		PosSide:    FirstLetterUpper(signal.OrderID),
		OrderType:  "market",
		Quantity:   quantity,
		Status:     model.OrderExecutionStatusPending,
	}

	if err := orderRepo.CreateWithAutoLog(ctx, newOrder); err != nil {
		logger.WithError(err).Error("failed to create order with auto log")
		return err
	}

	logger.WithField("order_id", newOrder.ID).Info("new kucoin order created")

	if err := kucoinClient.CloseAllPositions(newOrder.Symbol); err != nil {
		logger.WithError(err).
			WithField("symbol", newOrder.Symbol).
			Error("failed to close all positions on KuCoin")

		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to close existing positions",
		)

		return err
	}

	size := int64(quantity)
	if size <= 0 {
		size = 1
	}

	side := strings.ToLower(newOrder.Side)
	resp, err := kucoinClient.PlaceFuturesMarketOrder(
		newOrder.Symbol,
		side,
		size,
		false,
	)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"symbol": newOrder.Symbol,
			"side":   side,
			"qty":    size,
		}).WithError(err).Error("failed to place order on KuCoin")

		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKucoin",
			"controller",
			"kucoinClient.PlaceFuturesMarketOrder",
			"error",
			err,
			map[string]interface{}{
				"symbol": newOrder.Symbol,
				"side":   side,
				"qty":    size,
			},
		)
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to place order on KuCoin",
		)

		return err
	}

	var payload model.KucoinOrderResponse

	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		logger.WithFields(map[string]interface{}{
			"symbol": newOrder.Symbol,
		}).WithError(err).Error("failed to unmarshal kucoin response payload")

		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to decode kucoin response",
		)

		return err
	}

	ord, err := mapper.MapKucoinResponseToModel(&payload, newOrder.ID)
	if err != nil {
		logger.WithError(err).Error("failed to map kucoin response to model")

		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKucoin",
			"controller",
			"mapper.MapKucoinResponseToModel",
			"error",
			err,
			map[string]interface{}{},
		)
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to map kucoin response to model",
		)

		return err
	}

	if err := kucoinRepo.Create(ctx, ord); err != nil {
		logger.WithError(err).Error("failed to persist kucoin order")

		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKucoin",
			"controller",
			"kucoinRepo.Create",
			"error",
			err,
			map[string]interface{}{
				"symbol": newOrder.Symbol,
				"side":   side,
				"qty":    size,
			},
		)
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to persist kucoin order",
		)

		return err
	} else {
		if err := orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusPending, "order placed on KuCoin successfully"); err != nil {
		}
	}

	rawJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		logger.WithError(err).Error("failed to marshal kucoin raw response for storage")
		_ = orderRepo.UpdateStatusWithAutoLog(
			ctx,
			newOrder.ID,
			model.OrderExecutionStatusError,
			"failed to persist kucoin order",
		)
		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKucoin",
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

	if err := orderRepo.UpdateStatusWithAutoLog(
		ctx,
		newOrder.ID,
		model.OrderExecutionStatusFilled,
		"order executed successfully on kucoin",
	); err != nil {
		logger.WithError(err).Error("failed to update order final status")
		Capture(
			ctx,
			exceptionRepo,
			"OrderControllerKucoin",
			"controller",
			"orderRepo.UpdateStatusWithAutoLog",
			"error",
			err,
			map[string]interface{}{
				"symbol": newOrder.Symbol,
				"side":   side,
				"qty":    size,
			},
		)
		return err
	}

	logger.WithField("order_id", newOrder.ID).
		Info("order successfully completed on KuCoin")

	return nil
}
