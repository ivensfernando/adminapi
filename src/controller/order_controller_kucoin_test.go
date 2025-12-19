package controller

//import (
//	"context"
//	"errors"
//	"testing"
//
//	"strategyexecutor/src/externalmodel"
//	"strategyexecutor/src/model"
//)
//
//type mockKucoinClient struct {
//	baseSymbol  string
//	baseAvail   float64
//	usdtAvail   float64
//	price       float64
//	baseErr     error
//	convertSize int64
//	convertUSDT float64
//	convertErr  error
//	closeErr    error
//	orderResp   map[string]interface{}
//	orderErr    error
//}
//
//func (m *mockKucoinClient) GetAvailableBaseFromUSDT(symbol string) (string, float64, float64, float64, error) {
//	return m.baseSymbol, m.baseAvail, m.usdtAvail, m.price, m.baseErr
//}
//
//func (m *mockKucoinClient) GetFuturesAvailableFromRiskUnit(symbol string) (float64, error) {
//	return m.usdtAvail, m.baseErr
//}
//
//func (m *mockKucoinClient) ConvertUSDTToContracts(symbol string, usdt float64, leverage int) (int64, float64, error) {
//	return m.convertSize, m.convertUSDT, m.convertErr
//}
//
//func (m *mockKucoinClient) CloseAllPositions(symbol string) error {
//	return m.closeErr
//}
//
//func (m *mockKucoinClient) ExecuteFuturesOrderLeverage(symbol string, side string, orderType string, size int64, price *float64, leverage int, reduceOnly bool) (map[string]interface{}, error) {
//	return m.orderResp, m.orderErr
//}
//
//type mockKucoinOrderRepo struct {
//	created []*model.KucoinOrder
//	err     error
//}
//
//func (m *mockKucoinOrderRepo) Create(ctx context.Context, order *model.KucoinOrder) error {
//	if m.err != nil {
//		return m.err
//	}
//	m.created = append(m.created, order)
//	return nil
//}
//
//// Test successful flow end-to-end
//func TestOrderControllerKucoin_Success(t *testing.T) {
//	ctx := context.Background()
//	mTrading := &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 1, Symbol: "BTCUSDT", Action: "buy", OrderID: "long"}}}
//	mOrder := &mockOrderRepo{}
//	mKucoinOrder := &mockKucoinOrderRepo{}
//	mClient := &mockKucoinClient{
//		baseSymbol:  "BTCUSDT",
//		baseAvail:   1,
//		usdtAvail:   100,
//		price:       50000,
//		convertSize: 2,
//		convertUSDT: 50,
//		orderResp: map[string]interface{}{
//			"orderId":   "abc",
//			"clientOid": "client-1",
//			"symbol":    "BTCUSDTM",
//			"type":      "market",
//			"side":      "buy",
//			"price":     "50000",
//			"size":      "2",
//			"dealSize":  "2",
//			"dealValue": "100000",
//			"status":    "done",
//		},
//	}
//
//	oldTrading := newTradingSignalRepo
//	oldKucoinRepo := newKucoinOrderRepo
//	oldOrderRepo := newOrderRepo
//	oldException := newExceptionRepo
//
//	newTradingSignalRepo = func() tradingSignalRepository { return mTrading }
//	newKucoinOrderRepo = func() kucoinOrderRepository { return mKucoinOrder }
//	newOrderRepo = func() orderRepository { return mOrder }
//	newExceptionRepo = func() exceptionRepository { return &mockExceptionRepo{} }
//
//	defer func() {
//		newTradingSignalRepo = oldTrading
//		newKucoinOrderRepo = oldKucoinRepo
//		newOrderRepo = oldOrderRepo
//		newExceptionRepo = oldException
//	}()
//
//	user := &model.User{ID: 10, Username: "alice"}
//	if err := OrderControllerKucoin(ctx, mClient, user, 50, 2, "BTCUSDT", "kucoin"); err != nil {
//		t.Fatalf("expected success, got %v", err)
//	}
//
//	if len(mKucoinOrder.created) != 1 {
//		t.Fatalf("expected kucoin order to be created")
//	}
//	if mOrder.order == nil || mOrder.order.Symbol != "XBTUSDTM" {
//		t.Fatalf("expected mapped kucoin symbol, got %#v", mOrder.order)
//	}
//	if len(mOrder.statuses) == 0 || mOrder.statuses[len(mOrder.statuses)-1] != model.OrderExecutionStatusFilled {
//		t.Fatalf("expected final status filled, got %#v", mOrder.statuses)
//	}
//}
//
//func TestMapToKucoinFuturesSymbol(t *testing.T) {
//	cases := map[string]string{
//		"BTCUSD":   "XBTUSDTM",
//		"btcUsdt":  "XBTUSDTM",
//		"ETHUSDT":  "ETHUSDTM",
//		"SOLUSDTM": "SOLUSDTM",
//	}
//
//	for input, expected := range cases {
//		if got := mapToKucoinFuturesSymbol(input); got != expected {
//			t.Fatalf("expected %s to map to %s, got %s", input, expected, got)
//		}
//	}
//}
//
//// Test when trading signal repository fails
//func TestOrderControllerKucoin_SignalError(t *testing.T) {
//	ctx := context.Background()
//	mTrading := &mockTradingSignalRepo{err: errors.New("signal err")}
//	oldTrading := newTradingSignalRepo
//	newTradingSignalRepo = func() tradingSignalRepository { return mTrading }
//	defer func() { newTradingSignalRepo = oldTrading }()
//	if err := OrderControllerKucoin(ctx, &mockKucoinClient{}, &model.User{ID: 1}, 10, 1, "BTCUSDT", "kucoin"); err == nil {
//		t.Fatalf("expected error")
//	}
//}
//
//// Test when no trading signals are returned
//func TestOrderControllerKucoin_NoSignals(t *testing.T) {
//	ctx := context.Background()
//	mTrading := &mockTradingSignalRepo{}
//	oldTrading := newTradingSignalRepo
//	newTradingSignalRepo = func() tradingSignalRepository { return mTrading }
//	defer func() { newTradingSignalRepo = oldTrading }()
//
//	if err := OrderControllerKucoin(ctx, &mockKucoinClient{}, &model.User{ID: 1}, 10, 1, "BTCUSDT", "kucoin"); err != nil {
//		t.Fatalf("expected nil, got %v", err)
//	}
//}
//
//// Test close position failure propagates
//func TestOrderControllerKucoin_CloseFail(t *testing.T) {
//	ctx := context.Background()
//	mTrading := &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 1, Symbol: "BTCUSDT", Action: "buy"}}}
//	mOrder := &mockOrderRepo{}
//	oldTrading := newTradingSignalRepo
//	oldOrder := newOrderRepo
//	newTradingSignalRepo = func() tradingSignalRepository { return mTrading }
//	newOrderRepo = func() orderRepository { return mOrder }
//	defer func() {
//		newTradingSignalRepo = oldTrading
//		newOrderRepo = oldOrder
//	}()
//
//	mClient := &mockKucoinClient{usdtAvail: 10, price: 1, convertSize: 1, closeErr: errors.New("close")}
//
//	if err := OrderControllerKucoin(ctx, mClient, &model.User{ID: 1}, 10, 1, "BTCUSDT", "kucoin"); err == nil {
//		t.Fatalf("expected error from close positions")
//	}
//}
//
//// Test execute order failure propagates
//func TestOrderControllerKucoin_OrderFail(t *testing.T) {
//	ctx := context.Background()
//	mTrading := &mockTradingSignalRepo{signals: []externalmodel.TradingSignal{{ID: 1, Symbol: "BTCUSDT", Action: "buy"}}}
//	mOrder := &mockOrderRepo{}
//	oldTrading := newTradingSignalRepo
//	oldOrder := newOrderRepo
//	newTradingSignalRepo = func() tradingSignalRepository { return mTrading }
//	newOrderRepo = func() orderRepository { return mOrder }
//	defer func() {
//		newTradingSignalRepo = oldTrading
//		newOrderRepo = oldOrder
//	}()
//
//	mClient := &mockKucoinClient{usdtAvail: 10, price: 1, convertSize: 1, orderErr: errors.New("place")}
//
//	if err := OrderControllerKucoin(ctx, mClient, &model.User{ID: 1}, 10, 1, "BTCUSDT", "kucoin"); err == nil {
//		t.Fatalf("expected error from execute order")
//	}
//}
