package mapper

//import (
//	"testing"
//	"time"
//
//	"strategyexecutor/src/model"
//)
//
//func TestMapKucoinResponseToModel(t *testing.T) {
//	resp := &model.KucoinOrderResponse{
//		OrderID:     "order-123",
//		ClientOid:   "client-abc",
//		Symbol:      "XBTUSDTM",
//		Type:        "market",
//		Side:        "buy",
//		Price:       "25000",
//		Size:        "2",
//		DealSize:    "1",
//		DealValue:   "25000",
//		Leverage:    "10",
//		Fee:         "0.1",
//		FeeCurrency: "USDT",
//		TimeInForce: "IOC",
//		Remark:      "test",
//		OrderTime:   time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
//		Status:      "filled",
//	}
//
//	modelOrder, err := MapKucoinResponseToModel(resp, 42)
//	if err != nil {
//		t.Fatalf("expected no error mapping response, got %v", err)
//	}
//
//	if modelOrder == nil {
//		t.Fatalf("expected mapped order, got nil")
//	}
//
//	if modelOrder.OrderID != 42 || modelOrder.ExchangeOrderID != "order-123" {
//		t.Fatalf("unexpected order linkage: %+v", modelOrder)
//	}
//
//	if modelOrder.Price != 25000 || modelOrder.Size != 2 || modelOrder.FilledSize != 1 || modelOrder.Fee != 0.1 {
//		t.Fatalf("numeric fields not parsed correctly: %+v", modelOrder)
//	}
//
//	expectedTime := time.UnixMilli(resp.OrderTime)
//	if !modelOrder.OrderTime.Equal(expectedTime) {
//		t.Fatalf("expected order time %v, got %v", expectedTime, modelOrder.OrderTime)
//	}
//}
