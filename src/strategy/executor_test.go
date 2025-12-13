package strategy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"adminapi/src/model"
)

type stubConnector struct {
	executed []struct {
		OrderType string
		Symbol    string
		Quantity  float64
		Price     float64
	}

	executeErr error
}

func (s *stubConnector) TestConnection() error {
	return nil
}

func (s *stubConnector) GetAccountBalances() (map[string]float64, error) {
	return map[string]float64{}, nil
}

func (s *stubConnector) ExecuteOrder(orderType string, symbol string, quantity float64, price float64) (string, error) {
	if s.executeErr != nil {
		return "", s.executeErr
	}

	s.executed = append(s.executed, struct {
		OrderType string
		Symbol    string
		Quantity  float64
		Price     float64
	}{orderType, symbol, quantity, price})

	return "external-123", nil
}

func TestExecutorExecutesActions(t *testing.T) {
	logger, hook := logrustest.NewNullLogger()
	priceFromAlert := 102.5
	quantityFromAlert := 3.0

	executor := NewExecutor(logrus.NewEntry(logger), StaticConnectorProvider{
		1: &stubConnector{},
	})

	strat := &model.Strategy{
		ID:           10,
		UserID:       20,
		WebhookID:    30,
		Active:       true,
		ApplyOnAlert: true,
		Actions: []model.StrategyAction{{
			ID:            2,
			StrategyID:    10,
			ExchangeID:    1,
			Name:          "Buy BTC",
			ActionType:    "buy",
			OrderType:     "market",
			Symbol:        "BTCUSDT",
			Quantity:      0,
			StopLossPct:   1.0,
			TakeProfitPct: 2.0,
		}},
	}

	alert := &model.WebhookAlert{
		ID:       50,
		Price:    priceFromAlert,
		Quantity: quantityFromAlert,
		Ticker:   "BTCUSDT",
	}

	result := executor.Execute(context.Background(), strat, alert)

	if len(result.Errors) != 0 {
		t.Fatalf("expected no execution errors, got %d", len(result.Errors))
	}

	if len(result.Orders) != 1 {
		t.Fatalf("expected one order to be produced, got %d", len(result.Orders))
	}

	order := result.Orders[0]
	if order.Status != model.OrderStatusExecuted {
		t.Fatalf("expected order status %s, got %s", model.OrderStatusExecuted, order.Status)
	}

	if order.Price == nil || *order.Price != priceFromAlert {
		t.Fatalf("expected order price to use alert price %.2f, got %v", priceFromAlert, order.Price)
	}

	if order.Quantity != quantityFromAlert {
		t.Fatalf("expected order quantity %.2f, got %.2f", quantityFromAlert, order.Quantity)
	}

	if order.TriggeredByAlertID == nil || *order.TriggeredByAlertID != alert.ID {
		t.Fatalf("expected order to reference alert id %d", alert.ID)
	}

	if len(result.Logs) == 0 {
		t.Fatalf("expected at least one transaction log entry")
	}

	if len(hook.AllEntries()) == 0 {
		t.Fatalf("expected executor to emit logrus entries")
	}
}

func TestExecutorHandlesConnectorErrors(t *testing.T) {
	failingConnector := &stubConnector{executeErr: errors.New("boom")}
	executor := NewExecutor(logrus.NewEntry(logrus.StandardLogger()), StaticConnectorProvider{
		99: failingConnector,
	})

	strat := &model.Strategy{
		ID:           1,
		UserID:       2,
		WebhookID:    3,
		Active:       true,
		ApplyOnAlert: true,
		Actions: []model.StrategyAction{{
			ID:            9,
			StrategyID:    1,
			ExchangeID:    99,
			Name:          "Failing",
			ActionType:    "sell",
			OrderType:     "limit",
			Symbol:        "ETHUSDT",
			Quantity:      1,
			Price:         func() *float64 { v := 2000.0; return &v }(),
			StopLossPct:   1.0,
			TakeProfitPct: 2.0,
		}},
	}

	result := executor.Execute(context.Background(), strat, nil)

	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %d", len(result.Errors))
	}

	if len(result.Orders) != 1 {
		t.Fatalf("expected one order in results even on failure, got %d", len(result.Orders))
	}

	order := result.Orders[0]
	if order.Status != model.OrderStatusFailed {
		t.Fatalf("expected failed order status, got %s", order.Status)
	}

	if len(result.Logs) == 0 || result.Logs[0].Level != "error" {
		t.Fatalf("expected error transaction log")
	}
}

func TestExecutorUpdatesLastExecutedAt(t *testing.T) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	executor := NewExecutor(logger, StaticConnectorProvider{
		1: &stubConnector{},
	})

	strat := &model.Strategy{
		ID:           11,
		UserID:       12,
		WebhookID:    13,
		Active:       true,
		ApplyOnAlert: true,
		Actions: []model.StrategyAction{{
			ID:            100,
			StrategyID:    11,
			ExchangeID:    1,
			Name:          "Close",
			ActionType:    "close",
			OrderType:     "market",
			Symbol:        "ADAUSDT",
			Quantity:      5,
			StopLossPct:   1.0,
			TakeProfitPct: 2.0,
		}},
	}

	before := time.Now()
	result := executor.Execute(context.Background(), strat, nil)

	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(result.Errors))
	}

	if strat.LastExecutedAt == nil {
		t.Fatalf("expected LastExecutedAt to be set")
	}

	if strat.LastExecutedAt.Before(before) {
		t.Fatalf("expected LastExecutedAt to be updated after execution")
	}
}

func TestExecutorValidatesRiskControls(t *testing.T) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	executor := NewExecutor(logger, StaticConnectorProvider{
		1: &stubConnector{},
	})

	strat := &model.Strategy{
		ID:           21,
		UserID:       22,
		WebhookID:    23,
		Active:       true,
		ApplyOnAlert: true,
		Actions: []model.StrategyAction{{
			ID:            101,
			StrategyID:    21,
			ExchangeID:    1,
			Name:          "Missing stops",
			ActionType:    "buy",
			OrderType:     "market",
			Symbol:        "SOLUSDT",
			Quantity:      1,
			StopLossPct:   0,
			TakeProfitPct: 0,
		}},
	}

	result := executor.Execute(context.Background(), strat, nil)

	if len(result.Errors) != 1 {
		t.Fatalf("expected one validation error, got %d", len(result.Errors))
	}

	if len(result.Orders) != 0 {
		t.Fatalf("expected no orders when risk controls are missing")
	}
}

func TestExecutorCapsQuantityByMax(t *testing.T) {
	stub := &stubConnector{}
	executor := NewExecutor(logrus.NewEntry(logrus.StandardLogger()), StaticConnectorProvider{
		1: stub,
	})

	strat := &model.Strategy{
		ID:           31,
		UserID:       32,
		WebhookID:    33,
		Active:       true,
		ApplyOnAlert: true,
		Actions: []model.StrategyAction{{
			ID:            1,
			StrategyID:    31,
			ExchangeID:    1,
			Name:          "Cap quantity",
			ActionType:    "buy",
			OrderType:     "market",
			Symbol:        "XRPUSDT",
			Quantity:      0,
			MaxQuantity:   2,
			StopLossPct:   1.5,
			TakeProfitPct: 3.0,
		}},
	}

	alert := &model.WebhookAlert{ID: 99, Quantity: 5, Ticker: "XRPUSDT"}
	result := executor.Execute(context.Background(), strat, alert)

	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(result.Errors))
	}

	if len(result.Orders) != 1 {
		t.Fatalf("expected one order, got %d", len(result.Orders))
	}

	if result.Orders[0].Quantity != 2 {
		t.Fatalf("expected quantity to be capped at 2, got %.2f", result.Orders[0].Quantity)
	}

	if len(stub.executed) != 1 || stub.executed[0].Quantity != 2 {
		t.Fatalf("expected connector to execute capped quantity, got %+v", stub.executed)
	}
}
