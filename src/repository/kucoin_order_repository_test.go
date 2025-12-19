package repository

//import (
//	"context"
//	"regexp"
//	"testing"
//
//	"adminapi/src/model"
//
//	"github.com/DATA-DOG/go-sqlmock"
//)
//
//func TestKucoinOrderRepositoryQueries(t *testing.T) {
//	db, mock := newMockDB(t)
//	repo := (&KucoinOrderRepository{}).WithDB(db)
//
//	order := &model.KucoinOrder{
//		OrderID:         99,
//		ExchangeOrderID: "ex-123",
//		ClientOid:       "client-1",
//		Symbol:          "XBTUSDTM",
//		Side:            "buy",
//		OrderType:       "market",
//		Status:          "open",
//		Price:           25000,
//		Size:            2,
//		FilledSize:      1,
//		FilledValue:     25000,
//		Leverage:        10,
//		Fee:             0.1,
//		FeeCurrency:     "USDT",
//		TimeInForce:     "IOC",
//		Remark:          "test",
//	}
//
//	mock.ExpectBegin()
//	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "kucoin_orders" (`)).
//		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
//	mock.ExpectCommit()
//
//	if err := repo.Create(context.Background(), order); err != nil {
//		t.Fatalf("expected create to succeed, got %v", err)
//	}
//
//	row := sqlmock.NewRows([]string{"id", "order_id", "exchange_order_id", "client_oid", "symbol", "side", "order_type", "status", "price", "size", "filled_size", "filled_value", "leverage", "fee", "fee_currency", "time_in_force", "remark", "order_time", "created_at"}).
//		AddRow(1, order.OrderID, order.ExchangeOrderID, order.ClientOid, order.Symbol, order.Side, order.OrderType, order.Status, order.Price, order.Size, order.FilledSize, order.FilledValue, order.Leverage, order.Fee, order.FeeCurrency, order.TimeInForce, order.Remark, order.OrderTime, order.CreatedAt)
//
//	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "kucoin_orders" WHERE "kucoin_orders"."id" = $1 ORDER BY "kucoin_orders"."id" LIMIT $2`)).
//		WithArgs(uint(1), 1).
//		WillReturnRows(row)
//
//	found, err := repo.FindByID(context.Background(), 1)
//	if err != nil || found == nil {
//		t.Fatalf("expected to find record by id, got %+v err=%v", found, err)
//	}
//
//	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "kucoin_orders" WHERE order_id = $1 ORDER BY "kucoin_orders"."id" LIMIT $2`)).
//		WithArgs(order.ExchangeOrderID, 1).
//		WillReturnRows(row)
//
//	foundByOrder, err := repo.FindByOrderID(context.Background(), order.ExchangeOrderID)
//	if err != nil || foundByOrder == nil {
//		t.Fatalf("expected to find record by exchange order id, got %+v err=%v", foundByOrder, err)
//	}
//
//	latestRows := sqlmock.NewRows([]string{"id"}).AddRow(2).AddRow(1)
//	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "kucoin_orders" ORDER BY id DESC LIMIT $1`)).
//		WithArgs(20).
//		WillReturnRows(latestRows)
//
//	if _, err := repo.FindLatest(context.Background(), 0); err != nil {
//		t.Fatalf("expected FindLatest to succeed, got %v", err)
//	}
//
//	if err := mock.ExpectationsWereMet(); err != nil {
//		t.Fatalf("unmet expectations: %v", err)
//	}
//}
