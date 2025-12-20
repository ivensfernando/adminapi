package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"strategyexecutor/src/connectors"
	"strategyexecutor/src/externalmodel"
	"strategyexecutor/src/model"
	"strategyexecutor/src/tp_sl"
)

type mockTradingSignalRepo struct {
	signals []externalmodel.TradingSignal
	err     error
}

func (m *mockTradingSignalRepo) FindLatest(ctx context.Context, symbol, exchangeName string, limit int) ([]externalmodel.TradingSignal, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.signals, nil
}

type mockPhemexOrderRepo struct {
	created []*model.PhemexOrder
	err     error
}

func (m *mockPhemexOrderRepo) Create(ctx context.Context, order *model.PhemexOrder) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, order)
	return nil
}

type mockExceptionRepo struct{}

func (m *mockExceptionRepo) Create(ctx context.Context, exception *model.Exception) error {
	return nil
}

type mockOrderRepo struct {
	order          *model.Order
	findOrder      *model.Order
	findErr        error
	createErr      error
	updateErr      error
	updatePriceErr error
	updateRespErr  error
	statuses       []string
}

var _ orderRepository = (*mockOrderRepo)(nil)

func (m *mockOrderRepo) FindByExternalIDAndUserID(ctx context.Context, userID uint, externalID uint, orderDir string) (*model.Order, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.findOrder, nil
}

func (m *mockOrderRepo) CreateWithAutoLog(ctx context.Context, order *model.Order) error {
	if m.createErr != nil {
		return m.createErr
	}
	order.ID = 1
	m.order = order
	return nil
}

func (m *mockOrderRepo) UpdateStatusWithAutoLog(ctx context.Context, orderID uint, newStatus string, reason string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.statuses = append(m.statuses, newStatus)
	return nil
}

func (m *mockOrderRepo) UpdatePriceAutoLog(ctx context.Context, orderID uint, price *float64, reason string) error {
	if m.updatePriceErr != nil {
		return m.updatePriceErr
	}
	if m.order != nil && price != nil {
		m.order.Price = price
	}
	return nil
}

func (m *mockOrderRepo) UpdateResp(ctx context.Context, orderID uint, resp string, status string) error {
	if m.updateRespErr != nil {
		return m.updateRespErr
	}
	return nil
}

func (m *mockOrderRepo) UpdateStopLoss(ctx context.Context, orderID uint, stopLoss float64) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *mockOrderRepo) FindByExchangeIDAndUserID(ctx context.Context, userID uint, exchangeID uint) (*model.Order, error) {
	return nil, nil
}

type mockOHLCVRepo struct {
	newSL    decimal.Decimal
	isRaised bool
	err      error
}

func (m *mockOHLCVRepo) GetNextStopLoss(ctx context.Context, symbol string, now time.Time, side tp_sl.Side, currentSL decimal.Decimal, timeframe time.Duration, floor int) (decimal.Decimal, bool, error) {
	if m.err != nil {
		return decimal.Decimal{}, false, m.err
	}
	return m.newSL, m.isRaised, nil
}

type pos struct {
	AccountID        int64  `json:"accountID"`
	Symbol           string `json:"symbol"`
	Currency         string `json:"currency"`
	Side             string `json:"side"`
	PosSide          string `json:"posSide"`
	SizeRq           string `json:"sizeRq"`
	AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
	PositionMarginRv string `json:"positionMarginRv"`
	MarkPriceRp      string `json:"markPriceRp"`
}

type serverConfig struct {
	available         float64
	ticker            string
	positionsFirst    []pos
	positionsSecond   []pos
	positionsError    bool
	riskUnitError     bool
	tickerError       bool
	closeOrderError   bool
	placeOrderError   bool
	placeOrderNonZero bool
	placeOrderBadJSON bool
}

func convertPositions(ps []pos) []struct {
	AccountID        int64  `json:"accountID"`
	Symbol           string `json:"symbol"`
	Currency         string `json:"currency"`
	Side             string `json:"side"`
	PosSide          string `json:"posSide"`
	SizeRq           string `json:"sizeRq"`
	AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
	PositionMarginRv string `json:"positionMarginRv"`
	MarkPriceRp      string `json:"markPriceRp"`
} {
	converted := make([]struct {
		AccountID        int64  `json:"accountID"`
		Symbol           string `json:"symbol"`
		Currency         string `json:"currency"`
		Side             string `json:"side"`
		PosSide          string `json:"posSide"`
		SizeRq           string `json:"sizeRq"`
		AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
		PositionMarginRv string `json:"positionMarginRv"`
		MarkPriceRp      string `json:"markPriceRp"`
	}, 0, len(ps))

	for _, p := range ps {
		converted = append(converted, struct {
			AccountID        int64  `json:"accountID"`
			Symbol           string `json:"symbol"`
			Currency         string `json:"currency"`
			Side             string `json:"side"`
			PosSide          string `json:"posSide"`
			SizeRq           string `json:"sizeRq"`
			AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
			PositionMarginRv string `json:"positionMarginRv"`
			MarkPriceRp      string `json:"markPriceRp"`
		}{
			AccountID:        p.AccountID,
			Symbol:           p.Symbol,
			Currency:         p.Currency,
			Side:             p.Side,
			PosSide:          p.PosSide,
			SizeRq:           p.SizeRq,
			AvgEntryPriceRp:  p.AvgEntryPriceRp,
			PositionMarginRv: p.PositionMarginRv,
			MarkPriceRp:      p.MarkPriceRp,
		})
	}

	return converted
}

func buildPhemexTestClient(t *testing.T, cfg serverConfig) *connectors.Client {
	t.Helper()

	positionCalls := 0
	orderCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/risk-unit":
			if cfg.riskUnitError {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(connectors.APIResponse{Code: 0, Data: mustJSON([]connectors.RiskUnit{{
				Symbol:                "BTCUSDT",
				EstAvailableBalanceRv: cfg.available,
			}})})
		case "/md/v3/ticker/24hr":
			if cfg.tickerError {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"result": map[string]string{"lastRp": cfg.ticker}})
		case "/g-accounts/positions":
			if cfg.positionsError {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			positionCalls++
			positions := cfg.positionsFirst
			if positionCalls > 1 && len(cfg.positionsSecond) > 0 {
				positions = cfg.positionsSecond
			}
			_ = json.NewEncoder(w).Encode(connectors.APIResponse{Code: 0, Data: mustJSON(connectors.GAccountPositions{Positions: convertPositions(positions)})})
		case "/g-orders":
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			orderCalls++
			if cfg.closeOrderError {
				var payload map[string]interface{}
				_ = json.Unmarshal(bodyBytes, &payload)
				if reduceOnly, ok := payload["reduceOnly"].(bool); ok && reduceOnly {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			if orderCalls == 1 && cfg.closeOrderError {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if orderCalls > 1 {
				if cfg.placeOrderError {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if cfg.placeOrderBadJSON {
					_, _ = w.Write([]byte("not-json"))
					return
				}
				if cfg.placeOrderNonZero {
					_ = json.NewEncoder(w).Encode(connectors.APIResponse{Code: 500, Msg: "bad"})
					return
				}
			}
			_ = json.NewEncoder(w).Encode(connectors.APIResponse{Code: 0, Data: mustJSON(model.PhemexOrderResponse{OrderID: "abc", ClOrdID: "1", Symbol: "BTCUSDT", Side: "Buy", PriceRp: "50000", OrderQtyRq: "0.002"})})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return connectors.NewClient("k", "s", server.URL)
}

// TestOrderControllerFlows exercises Phemex order controller scenarios to ensure signals and orders
// are handled correctly across error and success paths.
func TestOrderControllerFlows(t *testing.T) {
	// Table-driven scenarios covering the most important branches of the
	// order execution flow for the Phemex exchange.
	tests := []struct {
		name                  string
		tradingRepo           *mockTradingSignalRepo
		orderRepo             *mockOrderRepo
		phemexRepo            *mockPhemexOrderRepo
		exceptionRepo         *mockExceptionRepo
		ohlcvRepo             *mockOHLCVRepo
		client                *connectors.Client
		expectError           bool
		expectOrder           bool
		expectedStatus        []string
		expectedPhemexCreates int
	}{
		{
			// success flow validates that a new order is created and
			// moves from pending to filled when everything succeeds.
			name:                  "success flow",
			tradingRepo:           &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:             &mockOrderRepo{},
			phemexRepo:            &mockPhemexOrderRepo{},
			exceptionRepo:         &mockExceptionRepo{},
			ohlcvRepo:             &mockOHLCVRepo{isRaised: false},
			client:                buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000", positionsFirst: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}, positionsSecond: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}}),
			expectOrder:           true,
			expectedStatus:        []string{model.OrderExecutionStatusPending, model.OrderExecutionStatusFilled},
			expectedPhemexCreates: 1,
		},
		{
			// trading signal repo error ensures repository failures are
			// propagated back to the caller.
			name:          "trading signal repo error",
			tradingRepo:   &mockTradingSignalRepo{err: errors.New("fail")},
			orderRepo:     &mockOrderRepo{},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectError:   true,
		},
		{
			// no signals returns nil confirms that absent signals do not
			// generate orders or errors.
			name:          "no signals returns nil",
			tradingRepo:   &mockTradingSignalRepo{},
			orderRepo:     &mockOrderRepo{},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectError:   false,
			expectOrder:   false,
		},
		{
			// existing order filled checks that already completed orders
			// are skipped without creating new ones.
			name:          "existing order filled",
			tradingRepo:   &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:     &mockOrderRepo{findOrder: &model.Order{ID: 99, Status: model.OrderExecutionStatusFilled}},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectOrder:   false,
		},
		{
			// find order error verifies failures when fetching existing
			// orders are surfaced.
			name:          "find order error",
			tradingRepo:   &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:     &mockOrderRepo{findErr: errors.New("find err")},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectError:   true,
		},
		{
			// create order error ensures persistence failures during new
			// order creation are handled.
			name:          "create order error",
			tradingRepo:   &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:     &mockOrderRepo{createErr: errors.New("create err")},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectError:   true,
		},
		{
			// close positions error checks that failures when closing
			// existing positions put the order into an error state.
			name:           "close positions error",
			tradingRepo:    &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:      &mockOrderRepo{},
			phemexRepo:     &mockPhemexOrderRepo{},
			exceptionRepo:  &mockExceptionRepo{},
			client:         buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000", positionsFirst: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}, closeOrderError: true}),
			expectError:    true,
			expectedStatus: []string{model.OrderExecutionStatusError},
		},
		{
			// place order http error simulates HTTP errors from the
			// Phemex endpoint during placement.
			name:           "place order http error",
			tradingRepo:    &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:      &mockOrderRepo{},
			phemexRepo:     &mockPhemexOrderRepo{},
			exceptionRepo:  &mockExceptionRepo{},
			client:         buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000", positionsFirst: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}, positionsSecond: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "0"}}, placeOrderError: true}),
			expectError:    true,
			expectedStatus: []string{model.OrderExecutionStatusError},
		},
		{
			// place order non zero code asserts non-zero API codes are
			// treated as failures when placing new orders.
			name:           "place order non zero code",
			tradingRepo:    &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:      &mockOrderRepo{},
			phemexRepo:     &mockPhemexOrderRepo{},
			exceptionRepo:  &mockExceptionRepo{},
			client:         buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000", positionsFirst: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}, positionsSecond: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "0"}}, placeOrderNonZero: true}),
			expectError:    true,
			expectedStatus: []string{model.OrderExecutionStatusError},
		},
		{
			// place order bad json ensures malformed JSON responses are
			// surfaced as errors.
			name:           "place order bad json",
			tradingRepo:    &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:      &mockOrderRepo{},
			phemexRepo:     &mockPhemexOrderRepo{},
			exceptionRepo:  &mockExceptionRepo{},
			client:         buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000", positionsFirst: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}, positionsSecond: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "0"}}, placeOrderBadJSON: true}),
			expectError:    true,
			expectedStatus: []string{model.OrderExecutionStatusError},
		},
		{
			// phemex repo create error checks persistence failures when
			// saving the Phemex order response locally.
			name:           "phemex repo create error",
			tradingRepo:    &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:      &mockOrderRepo{},
			phemexRepo:     &mockPhemexOrderRepo{err: errors.New("persist fail")},
			exceptionRepo:  &mockExceptionRepo{},
			client:         buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000", positionsFirst: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "0"}}, positionsSecond: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "0"}}}),
			expectError:    true,
			expectedStatus: []string{model.OrderExecutionStatusError},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Swap repositories for the duration of the test case and
			// restore the originals afterward.
			originalTrading := newTradingSignalRepo
			originalPhemex := newPhemexOrderRepo
			originalException := newExceptionRepo
			originalOrder := newOrderRepo
			originalOHLCV := newOHLCVRepo
			defer func() {
				newTradingSignalRepo = originalTrading
				newPhemexOrderRepo = originalPhemex
				newExceptionRepo = originalException
				newOrderRepo = originalOrder
				newOHLCVRepo = originalOHLCV
			}()

			newTradingSignalRepo = func() tradingSignalRepository { return tc.tradingRepo }
			newPhemexOrderRepo = func() phemexOrderRepository { return tc.phemexRepo }
			newExceptionRepo = func() exceptionRepository { return tc.exceptionRepo }
			newOrderRepo = func() orderRepository { return tc.orderRepo }
			newOHLCVRepo = func() ohlcvRepository {
				if tc.ohlcvRepo != nil {
					return tc.ohlcvRepo
				}
				return &mockOHLCVRepo{}
			}

			user := &model.User{ID: 1, Username: "tester"}

			// Execute the controller logic with the configured test
			// client and capture any returned error for assertions.
			userExchange := &model.UserExchange{OrderSizePercent: 50}
			err := OrderController(context.Background(), tc.client, user, uint(1), "BTCUSDT", "phemex", userExchange)
			if tc.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate whether an order was created based on the scenario
			// expectations.
			if tc.expectOrder && tc.orderRepo.order == nil {
				t.Fatalf("expected order to be created")
			}
			if !tc.expectOrder && tc.orderRepo.order != nil && tc.expectError == false {
				t.Fatalf("did not expect order creation")
			}
			if len(tc.expectedStatus) > 0 {
				// Ensure the controller progressed through the expected
				// status transitions.
				if len(tc.orderRepo.statuses) != len(tc.expectedStatus) {
					t.Fatalf("expected statuses %v got %v", tc.expectedStatus, tc.orderRepo.statuses)
				}
				for i, st := range tc.expectedStatus {
					if tc.orderRepo.statuses[i] != st {
						t.Fatalf("status %d expected %s got %s", i, st, tc.orderRepo.statuses[i])
					}
				}
			}
			if tc.expectedPhemexCreates > 0 {
				// Confirm we persisted the expected number of Phemex
				// orders when successful flows complete.
				if len(tc.phemexRepo.created) != tc.expectedPhemexCreates {
					t.Fatalf("expected %d phemex orders, got %d", tc.expectedPhemexCreates, len(tc.phemexRepo.created))
				}
			}
		})
	}
}

func mustJSON(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
