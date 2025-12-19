package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strategyexecutor/src/externalmodel"
	"strategyexecutor/src/mapper"
	"strategyexecutor/src/risk"
	"strategyexecutor/src/tp_sl"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	logger "github.com/sirupsen/logrus"

	"strategyexecutor/src/connectors"
	"strategyexecutor/src/model"
	"strategyexecutor/src/repository"
)

type tradingSignalRepository interface {
	FindLatest(ctx context.Context, symbol, exchangeName string, limit int) ([]externalmodel.TradingSignal, error)
}

type phemexOrderRepository interface {
	Create(ctx context.Context, order *model.PhemexOrder) error
}

type exceptionRepository interface {
	Create(ctx context.Context, exception *model.Exception) error
}

type orderRepository interface {
	FindByExternalIDAndUserID(ctx context.Context, userID uint, externalID uint, orderDir string) (*model.Order, error)
	CreateWithAutoLog(ctx context.Context, order *model.Order) error
	UpdateStatusWithAutoLog(ctx context.Context, orderID uint, newStatus string, reason string) error
	UpdatePriceAutoLog(ctx context.Context, orderID uint, price *float64, reason string) error
	FindByExchangeIDAndUserID(ctx context.Context, userID uint, exchangeID uint) (*model.Order, error)
}

var (
	newTradingSignalRepo = func() tradingSignalRepository {
		return repository.NewTradingSignalRepository()
	}
	newPhemexOrderRepo = func() phemexOrderRepository {
		return repository.NewPhemexOrderRepository()
	}
	newExceptionRepo = func() exceptionRepository {
		return repository.NewExceptionRepository()
	}
	newOrderRepo = func() orderRepository {
		return repository.NewOrderRepository()
	}
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
	user *model.User,
	orderSizePercent int,
	exchangeID uint,
	targetSymbol string, // BTCUSD
	targetExchange string, // phemex
) error {

	logger.Debugf("OrderController INITIALIZED ")
	logger.Info("starting order controller flow")

	tradingSignalRepo := repository.NewTradingSignalRepository()
	phemexRepo := repository.NewPhemexOrderRepository()
	exceptionRepo := repository.NewExceptionRepository()
	orderRepo := repository.NewOrderRepository()
	ohlcvRepo := repository.NewOHLCVRepositoryRepository()

	// ------------------------------------------------------------------
	// 1) Fetch the latest TradingSignal (from read-only DB)
	// ------------------------------------------------------------------
	signals, err := tradingSignalRepo.FindLatest(ctx, targetSymbol, targetExchange, 1)
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
		"user":          user.Username,
		"signal_id":     signal.ID,
		"signal.Symbol": signal.Symbol,
		"symbol":        symbol,
		"action":        signal.Action,
	}).Info("latest trading signal fetched")

	// ------------------------------------------------------------------
	// 2) Check if an order already exists for this signal
	// ------------------------------------------------------------------

	existingOrder, err := orderRepo.FindByExternalIDAndUserID(ctx, user.ID, signal.ID, model.OrderDirectionEntry)
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

			// check if we can raise the SL
			logger.WithField("order_id", existingOrder.ID).
				Info("order already filled, will check if we can raise the SL")

			side := tp_sl.SideLong
			if existingOrder.PosSide == "Short" {
				side = tp_sl.SideShort
			}

			newSL, isRaised, err := ohlcvRepo.GetNextStopLoss(
				ctx,
				existingOrder.Symbol,
				time.Now(),
				side,
				decimal.NewFromFloat(existingOrder.StopLossPct),
				15*time.Minute, // compute SL on 15m structure
				45,             // floor average over last 45 bars
			)
			if err != nil {
				logger.WithError(err).Error("failed to GetNextStopLoss")
				return err
			}

			if !isRaised {
				logger.
					WithField("order_id", existingOrder.ID).
					WithField("stop_loss_pct", existingOrder.StopLossPct).
					Info("order SL already set, nothing to do")
				return nil
			}

			_, err = phemexClient.SetStopLossForOpenPosition(
				"BTCUSDT",
				"Long",
				newSL.String(),
				connectors.TriggerByMarkPrice,
				true)
			if err != nil {
				logger.WithError(err).Error("failed to SetStopLossForOpenPosition")
				return err
			}

			err = orderRepo.UpdateStopLoss(ctx, existingOrder.ID, newSL.InexactFloat64())
			if err != nil {
				logger.WithError(err).Error("failed to UpdateStopLoss")
				return err
			}

			// update SL

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

	cfg := risk.DefaultSessionSizeConfig()

	finalSize, session := risk.CalculateSizeByNYSession(
		decimal.NewFromFloat(value),
		time.Now(),
		cfg,
	)

	valueWithRisk := finalSize.InexactFloat64()

	logger.
		WithField("session", session).
		WithField("baseSize", value).
		WithField("finalSize", finalSize).
		WithField("valueWithRisk", valueWithRisk).
		WithField("Symbol", symbol).
		Info("session based risk sizing")

	// check if we can place the order based on risk settings
	if valueWithRisk <= 0 || session == risk.SessionNoTrade {
		logger.
			WithField("session", session).
			WithField("baseSize", value).
			WithField("finalSize", finalSize).
			WithField("valueWithRisk", valueWithRisk).
			WithField("Symbol", symbol).
			Warn("risk sizing - unable to place order due to risk settings")

		if err := phemexClient.CloseAllPositions(symbol); err != nil {
			logger.WithError(err).
				WithField("symbol", symbol).
				Error("failed to close all positions")
			return err
		}
		return nil
	}

	logger.
		WithField("session", session).
		WithField("baseSize", value).
		WithField("finalSize", finalSize).
		WithField("valueWithRisk", valueWithRisk).
		WithField("Symbol", symbol).
		Debug("Value of order in ")
	// ------------------------------------------------------------------
	// 3) Create new Order (Phemex = exchange_id 1)
	// ------------------------------------------------------------------

	newOrder := &model.Order{
		UserID:     user.ID,
		ExchangeID: exchangeID, // Phemex
		ExternalID: signal.ID,
		Symbol:     symbol,                           //signal.Symbol, "BTCUSDT"
		Side:       FirstLetterUpper(signal.Action),  // buy/sell
		PosSide:    FirstLetterUpper(signal.OrderID), //Short/Long
		OrderType:  "market",
		Quantity:   valueWithRisk, //
		Status:     model.OrderExecutionStatusPending,
		OrderDir:   model.OrderDirectionEntry,
	}

	if err := orderRepo.CreateWithAutoLog(ctx, newOrder); err != nil {
		logger.WithError(err).Error("failed to create order with auto log")
		return err
	}

	logger.WithField("order_id", newOrder.ID).Info("new order created")

	// ------------------------------------------------------------------
	// 4) Close all existing positions for this symbol on Phemex
	// ------------------------------------------------------------------

	if err := closeAllPositions(ctx, phemexClient, user, exchangeID, signal.ID, newOrder.Symbol); err != nil {
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

	// TODO: ADD STOP LOSS
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

	if err := orderRepo.UpdatePriceAutoLog(ctx, newOrder.ID, &ord.Price, "update to price phemex order"); err != nil {
		logger.WithError(err).Error("failed to update price on order")
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
	} else {
		if err := orderRepo.UpdateStatusWithAutoLog(ctx, newOrder.ID, model.OrderExecutionStatusPending, "order placed on Phemex successfully"); err != nil {
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

func closeAllPositions(
	ctx context.Context,
	phemexClient *connectors.Client,
	user *model.User,
	exchangeID uint,
	signalID uint,
	symbol string,
) error {

	phemexRepo := repository.NewPhemexOrderRepository()
	exceptionRepo := repository.NewExceptionRepository()
	orderRepo := repository.NewOrderRepository()

	logger.WithFields(map[string]interface{}{
		"symbol": symbol,
		"user":   user.Username,
	}).Info("Closing ALL positions for symbol")

	// 1) Fetch all USDT positions from the account
	positions, err := phemexClient.GetPositionsUSDT()
	if err != nil {
		return fmt.Errorf("GetPositionsUSDT failed: %w", err)
	}

	// 2) Iterate through positions and filter by symbol
	for _, p := range positions.Positions {
		if p.Symbol != symbol {
			continue
		}

		// Skip empty positions (nothing to close)
		if p.SizeRq == "0" || p.SizeRq == "" {
			continue
		}

		// Determine the opposite side required to close the position
		var closeSide string
		switch p.Side {
		case "Buy":
			closeSide = "Sell"
		case "Sell":
			closeSide = "Buy"
		default:
			logger.WithFields(map[string]interface{}{
				"symbol": symbol,
				"side":   p.Side,
			}).Error("Unknown position side, skipping")
			continue
		}

		quantity, err := strconv.ParseFloat(p.SizeRq, 64)
		if err != nil {
			logger.WithError(err).Error("failed to parse SizeRq to float")
			return err
		}

		exitOrder := &model.Order{
			UserID:     user.ID,
			ExchangeID: exchangeID, // Phemex
			ExternalID: signalID,
			Symbol:     symbol,    //signal.Symbol, "BTCUSDT"
			Side:       p.PosSide, // buy/sell
			PosSide:    closeSide, //Short/Long
			OrderType:  "market",
			Quantity:   quantity, //
			Status:     model.OrderExecutionStatusPending,
			OrderDir:   model.OrderDirectionExit,
		}

		if err := orderRepo.CreateWithAutoLog(ctx, exitOrder); err != nil {
			logger.WithError(err).Error("failed to create exit order with auto log")
			return err
		}

		logger.WithField("order_id", exitOrder.ID).Info("new exit order created")

		logger.WithFields(map[string]interface{}{
			"symbol":    p.Symbol,
			"posSide":   p.PosSide,
			"side":      p.Side,
			"size":      p.SizeRq,
			"closeSide": closeSide,
		}).Info("Closing position")

		// 3) Send a MARKET order with reduceOnly to fully close the position
		resp, err := phemexClient.PlaceOrder(
			p.Symbol,  // trading pair
			closeSide, // opposite side to close the position
			p.PosSide, // Long or Short
			p.SizeRq,  // full position size
			"Market",  // market order
			true,      // reduceOnly = true (guarantees position close)
		)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"symbol":  p.Symbol,
				"posSide": p.PosSide,
				"side":    p.Side,
				"size":    p.SizeRq,
			}).WithError(err).Error("Failed to close position")

			return fmt.Errorf(
				"failed to close position %s %s (%s): %w",
				p.Symbol,
				p.PosSide,
				p.Side,
				err,
			)
		}

		if resp.Code != 0 {
			logger.WithFields(map[string]interface{}{
				"symbol": p.Symbol,
				"code":   resp.Code,
				"msg":    resp.Msg,
			}).Error("Phemex returned non-zero code")

			_ = orderRepo.UpdateStatusWithAutoLog(
				ctx,
				exitOrder.ID,
				model.OrderExecutionStatusCanceledError,
				"phemex returned non-zero code while placing order",
			)

			return fmt.Errorf("phemex error %d: %s", resp.Code, resp.Msg)
		} else {
			if err := orderRepo.UpdateStatusWithAutoLog(
				ctx,
				exitOrder.ID,
				model.OrderExecutionStatusFilled,
				"order executed successfully on phemex",
			); err != nil {
				logger.WithError(err).Error("failed to update order final status")
				Capture(
					ctx,
					exceptionRepo,
					"OrderController closeAllPositions",
					"controller",
					"orderRepo.UpdateStatusWithAutoLog",
					"error",
					err,
					map[string]interface{}{
						"symbol": exitOrder.Symbol,
						"side":   exitOrder.Side,
						"qty":    quantity,
					},
				)
				return err
			}
		}

		var payload model.PhemexOrderResponse

		if err := json.Unmarshal(resp.Data, &payload); err != nil {
			logger.WithFields(map[string]interface{}{
				"symbol": p.Symbol,
			}).WithError(err).Error("closeAllPositions failed to unmarshal phemex response payload")
			return err
		}

		// Map API payload -> DB model (versão safe)
		ord, err := mapper.MapPhemexResponseToModel(&payload, exitOrder.ID)
		if err != nil {
			logger.WithError(err).Error("closeAllPositions failed to map phemex response to model")

			Capture(
				ctx,
				exceptionRepo,
				"OrderController closeAllPositions",
				"controller",
				"mapper.MapPhemexResponseToModel",
				"error",
				err,
				map[string]interface{}{},
			)

			return err
		}

		// Persist Phemex order in DB
		if err := phemexRepo.Create(ctx, ord); err != nil {
			logger.WithError(err).Error("closeAllPositions failed to persist phemex order")

			Capture(
				ctx,
				exceptionRepo,
				"OrderController closeAllPositions",
				"controller",
				"phemexRepo.Create",
				"error",
				err,
				map[string]interface{}{
					"symbol": p.Symbol,
					"side":   p.Side,
				},
			)

			return err
		}

	}

	return nil
}
