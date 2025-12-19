package controller

import (
	"context"
	"encoding/json"
	"github.com/shopspring/decimal"
	logger "github.com/sirupsen/logrus"
	"strategyexecutor/src/mapper"
	"strategyexecutor/src/model"
	"strategyexecutor/src/repository"
	"strategyexecutor/src/risk"
	"strings"
	"time"
)

type kucoinOrderRepository interface {
	Create(ctx context.Context, order *model.KucoinOrder) error
}

type kucoinFuturesClient interface {
	GetAvailableBaseFromUSDT(symbol string) (baseSymbol string, baseAvail float64, usdtAvail float64, price float64, err error)
	ConvertUSDTToContracts(symbol string, usdt float64, leverage int) (size int64, usdtUsed float64, err error)
	CloseAllPositions(symbol string) error
	ExecuteFuturesOrderLeverage(symbol string, side string, orderType string, size int64, price *float64, leverage int, reduceOnly bool) (map[string]interface{}, error)
	GetFuturesAvailableFromRiskUnit(symbol string) (float64, error)
}

var (
	newKucoinOrderRepo = func() kucoinOrderRepository {
		return repository.NewKucoinOrderRepository()
	}
)

// OrderControllerKucoin executes the trading flow for KuCoin using the latest signal.
func OrderControllerKucoin(
	ctx context.Context,
	kucoinClient kucoinFuturesClient,
	user *model.User,
	orderSizePercent int,
	exchangeID uint,
	targetSymbol string, // BTCUSD
	targetExchange string,
) error {

	logger.Debugf("OrderControllerKucoin INITIALIZED ")
	logger.Info("starting kucoin order controller flow")

	tradingSignalRepo := newTradingSignalRepo()
	kucoinRepo := newKucoinOrderRepo()
	exceptionRepo := newExceptionRepo()
	orderRepo := newOrderRepo()

	signals, err := tradingSignalRepo.FindLatest(ctx, targetSymbol, targetExchange, 1)
	if err != nil {
		Capture(ctx, exceptionRepo.(*repository.ExceptionRepository), "OrderControllerKucoin", "controller", "tradingSignalRepo.FindLatest", "error", err, map[string]interface{}{})
		return err
	}
	if len(signals) == 0 {
		logger.Warn("no trading signals found for kucoin")
		return nil
	}

	signal := signals[0]
	normalizedSymbol := NormalizeToUSDT(signal.Symbol)
	symbol := mapToKucoinFuturesSymbol(normalizedSymbol)
	logger.WithFields(map[string]interface{}{
		"user":      user.Username,
		"signal_id": signal.ID,
		"symbol":    symbol,
		"action":    signal.Action,
	}).Info("latest kucoin trading signal fetched")

	existingOrder, err := orderRepo.FindByExternalIDAndUserID(ctx, user.ID, signal.ID)
	if err != nil {
		Capture(ctx, exceptionRepo.(*repository.ExceptionRepository), "OrderControllerKucoin", "controller", "orderRepo.FindByExternalIDAndUser", "error", err, map[string]interface{}{})
		return err
	}

	if existingOrder != nil {
		if existingOrder.Status == model.OrderExecutionStatusFilled {
			logger.WithField("order_id", existingOrder.ID).Info("existing kucoin order already filled")
			return nil
		}
	}

	_, _, _, price, err := kucoinClient.GetAvailableBaseFromUSDT(symbol)
	if err != nil {
		Capture(ctx, exceptionRepo.(*repository.ExceptionRepository), "OrderControllerKucoin", "controller", "kucoinClient.GetAvailableBaseFromUSDT", "error", err, map[string]interface{}{"symbol": symbol})
		return err
	}

	usdtAvail, err := kucoinClient.GetFuturesAvailableFromRiskUnit(symbol)
	if err != nil {
		Capture(ctx, exceptionRepo.(*repository.ExceptionRepository), "OrderControllerKucoin", "controller", "kucoinClient.GetFuturesAvailableFromRiskUnit", "error", err, map[string]interface{}{"symbol": symbol})
		return err
	}

	value := PercentOfFloatSafe(usdtAvail, orderSizePercent)
	cfg := risk.DefaultSessionSizeConfig()
	finalSize, session := risk.CalculateSizeByNYSession(decimal.NewFromFloat(value), time.Now(), cfg)
	value = finalSize.InexactFloat64()

	logger.WithFields(map[string]interface{}{
		"usdt_available": usdtAvail,
		"price":          price,
		"session":        session,
		"usdt_value":     value,
	}).Info("kucoin risk sizing complete")

	contracts, usedUSDT, err := kucoinClient.ConvertUSDTToContracts(symbol, value, 1)
	if err != nil {
		Capture(ctx, exceptionRepo.(*repository.ExceptionRepository), "OrderControllerKucoin", "controller", "kucoinClient.ConvertUSDTToContracts", "error", err, map[string]interface{}{"symbol": symbol})
		return err
	}

	newOrder := &model.Order{
		UserID:     user.ID,
		ExchangeID: exchangeID,
		ExternalID: signal.ID,
		Symbol:     symbol,
		Side:       strings.ToLower(signal.Action),
		PosSide:    FirstLetterUpper(signal.OrderID),
		OrderType:  "market",
		Quantity:   float64(contracts),
		Price:      &price,
		Status:     model.OrderExecutionStatusPending,
	}

	if err := orderRepo.CreateWithAutoLog(ctx, newOrder); err != nil {
		return err
	}

	if err := kucoinClient.CloseAllPositions(newOrder.Symbol); err != nil {
		_ = orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusError, "failed to close existing positions on kucoin")
		return err
	}

	resp, err := kucoinClient.ExecuteFuturesOrderLeverage(newOrder.Symbol, newOrder.Side, "market", contracts, nil, 1, false)
	if err != nil {
		_ = orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusError, "failed to place kucoin futures order")
		return err
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		_ = orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusError, "failed to marshal kucoin response")
		return err
	}

	var payload model.KucoinOrderResponse
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		_ = orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusError, "failed to decode kucoin response")
		return err
	}

	mapped, err := mapper.MapKucoinResponseToModel(&payload, newOrder.ID)
	if err != nil {
		Capture(ctx, exceptionRepo.(*repository.ExceptionRepository), "OrderControllerKucoin", "controller", "mapper.MapKucoinResponseToModel", "error", err, map[string]interface{}{"symbol": symbol})
		_ = orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusError, "failed to map kucoin response")
		return err
	}

	if mapped != nil && mapped.Price > 0 {
		priceCopy := mapped.Price
		_ = orderRepo.UpdatePriceAutoLog(ctx, newOrder.ID, &priceCopy, "update to price kucoin order")
	}

	if err := kucoinRepo.Create(ctx, mapped); err != nil {
		Capture(ctx, exceptionRepo.(*repository.ExceptionRepository), "OrderControllerKucoin", "controller", "kucoinRepo.Create", "error", err, map[string]interface{}{"symbol": symbol})
		_ = orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusError, "failed to persist kucoin order")
		return err
	}

	_ = orderRepo.UpdateResp(ctx, newOrder.ID, string(respBytes), model.OrderExecutionStatusPending)
	_ = orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusFilled, "order executed successfully on kucoin")
	logger.WithFields(map[string]interface{}{"order_id": newOrder.ID, "used_usdt": usedUSDT}).Info("kucoin order executed successfully")

	return nil
}

func mapToKucoinFuturesSymbol(symbol string) string {
	upper := strings.ToUpper(symbol)

	base := upper
	switch {
	case strings.HasSuffix(upper, "USDTM"):
		base = strings.TrimSuffix(upper, "USDTM")
	case strings.HasSuffix(upper, "USDT"):
		base = strings.TrimSuffix(upper, "USDT")
	case strings.HasSuffix(upper, "USD"):
		base = strings.TrimSuffix(upper, "USD")
	}

	if base == "BTC" {
		base = "XBT"
	}

	return base + "USDTM"
}
