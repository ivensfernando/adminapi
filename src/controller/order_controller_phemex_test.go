package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"

	"strategyexecutor/src/connectors"
	"strategyexecutor/src/externalmodel"
	"strategyexecutor/src/model"
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

func (m *mockOrderRepo) FindByExternalIDAndUserID(ctx context.Context, userID uint, externalID uint) (*model.Order, error) {
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

func (m *mockOrderRepo) FindByExchangeIDAndUserID(ctx context.Context, userID uint, exchangeID uint) (*model.Order, error) {
	return nil, nil
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
			orderCalls++
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

	restyClient := resty.New().SetBaseURL(server.URL)
	restyClient.SetTransport(server.Client().Transport)

	return nil //connectors.NewClientWithResty("k", "s", server.URL, restyClient)
}

func TestOrderControllerFlows(t *testing.T) {
	tests := []struct {
		name                  string
		tradingRepo           *mockTradingSignalRepo
		orderRepo             *mockOrderRepo
		phemexRepo            *mockPhemexOrderRepo
		exceptionRepo         *mockExceptionRepo
		client                *connectors.Client
		expectError           bool
		expectOrder           bool
		expectedStatus        []string
		expectedPhemexCreates int
	}{
		{
			name:                  "success flow",
			tradingRepo:           &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:             &mockOrderRepo{},
			phemexRepo:            &mockPhemexOrderRepo{},
			exceptionRepo:         &mockExceptionRepo{},
			client:                buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000", positionsFirst: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}, positionsSecond: []pos{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}}),
			expectOrder:           true,
			expectedStatus:        []string{model.OrderExecutionStatusPending, model.OrderExecutionStatusFilled},
			expectedPhemexCreates: 1,
		},
		{
			name:          "trading signal repo error",
			tradingRepo:   &mockTradingSignalRepo{err: errors.New("fail")},
			orderRepo:     &mockOrderRepo{},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectError:   true,
		},
		{
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
			name:          "existing order filled",
			tradingRepo:   &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:     &mockOrderRepo{findOrder: &model.Order{ID: 99, Status: model.OrderExecutionStatusFilled}},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectOrder:   false,
		},
		{
			name:          "find order error",
			tradingRepo:   &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:     &mockOrderRepo{findErr: errors.New("find err")},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectError:   true,
		},
		{
			name:          "create order error",
			tradingRepo:   &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 10, OrderID: "long", Symbol: "BTCUSDT", Action: "buy", ExchangeName: "phemex"}}},
			orderRepo:     &mockOrderRepo{createErr: errors.New("create err")},
			phemexRepo:    &mockPhemexOrderRepo{},
			exceptionRepo: &mockExceptionRepo{},
			client:        buildPhemexTestClient(t, serverConfig{available: 100, ticker: "50000"}),
			expectError:   true,
		},
		{
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
			originalTrading := newTradingSignalRepo
			originalPhemex := newPhemexOrderRepo
			originalException := newExceptionRepo
			originalOrder := newOrderRepo
			defer func() {
				newTradingSignalRepo = originalTrading
				newPhemexOrderRepo = originalPhemex
				newExceptionRepo = originalException
				newOrderRepo = originalOrder
			}()

			newTradingSignalRepo = func() tradingSignalRepository { return tc.tradingRepo }
			newPhemexOrderRepo = func() phemexOrderRepository { return tc.phemexRepo }
			newExceptionRepo = func() exceptionRepository { return tc.exceptionRepo }
			newOrderRepo = func() orderRepository { return tc.orderRepo }

			user := &model.User{ID: 1, Username: "tester"}

			err := OrderController(context.Background(), tc.client, user, 50, 1, "BTCUSDT", "phemex")
			if tc.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectOrder && tc.orderRepo.order == nil {
				t.Fatalf("expected order to be created")
			}
			if !tc.expectOrder && tc.orderRepo.order != nil && tc.expectError == false {
				t.Fatalf("did not expect order creation")
			}
			if len(tc.expectedStatus) > 0 {
				if len(tc.orderRepo.statuses) != len(tc.expectedStatus) {
					t.Fatalf("expected statuses %v got %v", tc.expectedStatus, tc.orderRepo.statuses)
				}
				for i, st := range tc.expectedStatus {
					if tc.orderRepo.statuses[i] != st {
						t.Fatalf("status %d expected %s got %s", i, st, tc.orderRepo.statuses[i])
					}
				}
			}

			if tc.expectedPhemexCreates > 0 && len(tc.phemexRepo.created) != tc.expectedPhemexCreates {
				t.Fatalf("expected %d phemex orders, got %d", tc.expectedPhemexCreates, len(tc.phemexRepo.created))
			}
		})
	}
}

func mustJSON(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
