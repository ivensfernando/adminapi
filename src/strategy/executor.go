package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"adminapi/internal/connectors"
	"adminapi/src/model"
)

type ConnectorProvider interface {
	ConnectorForExchange(exchangeID uint) (connectors.ExchangeConnector, error)
}

type StaticConnectorProvider map[uint]connectors.ExchangeConnector

func (p StaticConnectorProvider) ConnectorForExchange(exchangeID uint) (connectors.ExchangeConnector, error) {
	connector, ok := p[exchangeID]
	if !ok {
		return nil, fmt.Errorf("connector for exchange %d not found", exchangeID)
	}

	return connector, nil
}

type Executor struct {
	logger   *logrus.Entry
	provider ConnectorProvider
	now      func() time.Time
}

func NewExecutor(logger *logrus.Entry, provider ConnectorProvider) *Executor {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}

	return &Executor{logger: logger, provider: provider, now: time.Now}
}

type ExecutionResult struct {
	Orders []model.Order
	Logs   []model.TransactionLog
	Errors []error
}

func (e *Executor) Execute(ctx context.Context, strat *model.Strategy, alert *model.WebhookAlert) ExecutionResult {
	if ctx == nil {
		ctx = context.Background()
	}

	result := ExecutionResult{}
	if strat == nil {
		err := fmt.Errorf("strategy is nil")
		result.Errors = append(result.Errors, err)
		return result
	}

	if !strat.Active || !strat.ApplyOnAlert {
		msg := "strategy skipped: inactive or not configured to auto-apply"
		e.logger.WithField("strategy_id", strat.ID).Info(msg)
		result.Logs = append(result.Logs, e.logEntry(strat, nil, nil, "info", msg, nil))
		return result
	}

	for i := range strat.Actions {
		select {
		case <-ctx.Done():
			err := fmt.Errorf("execution canceled: %w", ctx.Err())
			result.Errors = append(result.Errors, err)
			result.Logs = append(result.Logs, e.logEntry(strat, nil, nil, "warn", "execution canceled", map[string]any{"error": err.Error()}))
			return result
		default:
		}

		action := strat.Actions[i]

		if action.StopLossPct <= 0 || action.TakeProfitPct <= 0 {
			err := fmt.Errorf("strategy action missing stop-loss/take-profit configuration")
			result.Errors = append(result.Errors, err)
			result.Logs = append(result.Logs, e.logEntry(strat, &action, nil, "error", "action validation failed", map[string]any{"error": err.Error()}))
			continue
		}

		conn, err := e.provider.ConnectorForExchange(action.ExchangeID)
		if err != nil {
			e.logger.WithError(err).WithFields(logrus.Fields{
				"strategy_id": strat.ID,
				"action_id":   action.ID,
				"exchange_id": action.ExchangeID,
			}).Error("failed to resolve connector for action")

			result.Errors = append(result.Errors, err)
			result.Logs = append(result.Logs, e.logEntry(strat, &action, nil, "error", "failed to resolve connector", map[string]any{"error": err.Error()}))
			continue
		}

		order := e.buildOrder(strat, &action, alert)
		if order.Quantity <= 0 {
			err := fmt.Errorf("no executable quantity after risk limits")
			result.Errors = append(result.Errors, err)
			result.Logs = append(result.Logs, e.logEntry(strat, &action, &order, "error", "order validation failed", map[string]any{"error": err.Error()}))
			continue
		}
		price := 0.0
		if order.Price != nil {
			price = *order.Price
		}

		execID, execErr := conn.ExecuteOrder(order.OrderType, order.Symbol, order.Quantity, price)
		if execErr != nil {
			e.logger.WithError(execErr).WithFields(logrus.Fields{
				"strategy_id": strat.ID,
				"action_id":   action.ID,
			}).Error("failed to execute order")

			order.Status = model.OrderStatusFailed
			result.Errors = append(result.Errors, execErr)
			result.Logs = append(result.Logs, e.logEntry(strat, &action, &order, "error", "order execution failed", map[string]any{"error": execErr.Error()}))
			result.Orders = append(result.Orders, order)
			continue
		}

		now := e.now()
		order.Status = model.OrderStatusExecuted
		order.ExecutedAt = &now
		order.ExternalID = execID

		result.Orders = append(result.Orders, order)
		result.Logs = append(result.Logs, e.logEntry(strat, &action, &order, "info", "order executed", map[string]any{"execution_id": execID}))
		e.logger.WithFields(logrus.Fields{
			"strategy_id": strat.ID,
			"action_id":   action.ID,
			"order_id":    order.ID,
		}).Info("strategy action executed")
	}

	executedAt := e.now()
	strat.LastExecutedAt = &executedAt

	return result
}

func (e *Executor) buildOrder(strat *model.Strategy, action *model.StrategyAction, alert *model.WebhookAlert) model.Order {
	qty := action.Quantity

	if alert != nil && alert.Quantity != 0 {
		qty = alert.Quantity
	}

	if action.MaxQuantity > 0 && qty > action.MaxQuantity {
		qty = action.MaxQuantity
	}

	order := model.Order{
		StrategyActionID: action.ID,
		StrategyID:       strat.ID,
		UserID:           strat.UserID,
		ExchangeID:       action.ExchangeID,
		Symbol:           action.Symbol,
		Side:             action.ActionType,
		OrderType:        action.OrderType,
		Quantity:         qty,
		Price:            action.Price,
		StopLossPct:      action.StopLossPct,
		TakeProfitPct:    action.TakeProfitPct,
		Status:           model.OrderStatusPending,
	}

	if alert != nil {
		order.TriggeredByAlertID = &alert.ID

		if order.Price == nil && alert.Price != 0 {
			price := alert.Price
			order.Price = &price
		}

		if order.Quantity == 0 && alert.Quantity != 0 {
			order.Quantity = alert.Quantity
		}

		if order.Symbol == "" && alert.Ticker != "" {
			order.Symbol = alert.Ticker
		}
	}

	return order
}

func (e *Executor) logEntry(strat *model.Strategy, action *model.StrategyAction, order *model.Order, level string, message string, metadata map[string]any) model.TransactionLog {
	entry := model.TransactionLog{
		Level:     level,
		Message:   message,
		CreatedAt: e.now(),
	}

	if strat != nil {
		entry.StrategyID = &strat.ID
	}

	if action != nil {
		entry.StrategyActionID = &action.ID
	}

	if order != nil {
		entry.OrderID = &order.ID
	}

	if len(metadata) > 0 {
		entry.Metadata = metadata
	}

	return entry
}
