package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"adminapi/src/model"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestOrderRepositorySearch(t *testing.T) {
	mockDB, mock := newMockDB(t)
	repo := &OrderRepository{db: mockDB}

	createdAt := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	orders := []model.Order{
		{ID: 1, UserID: 1, ExchangeID: 1, Symbol: "BTCUSDT", CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: 2, UserID: 1, ExchangeID: 2, Symbol: "ETHUSDT", CreatedAt: createdAt.Add(24 * time.Hour), UpdatedAt: createdAt.Add(24 * time.Hour)},
		{ID: 3, UserID: 2, ExchangeID: 1, Symbol: "SOLUSDT", CreatedAt: createdAt.Add(48 * time.Hour), UpdatedAt: createdAt.Add(48 * time.Hour)},
	}

	orderRows := func(returned ...model.Order) *sqlmock.Rows {
		rows := sqlmock.NewRows([]string{"id", "user_id", "exchange_id", "symbol", "created_at", "updated_at"})
		for _, order := range returned {
			rows.AddRow(order.ID, order.UserID, order.ExchangeID, order.Symbol, order.CreatedAt, order.UpdatedAt)
		}
		return rows
	}

	t.Run("filters by user", func(t *testing.T) {
		mockRows := orderRows(orders[1], orders[0])
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE user_id = $1 ORDER BY created_at DESC, id DESC`)).
			WithArgs(uint(1)).
			WillReturnRows(mockRows)

		results, err := repo.Search(context.Background(), OrderSearchOptions{UserID: 1})
		if err != nil {
			t.Fatalf("unexpected error searching orders: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("expected 2 orders for user 1, got %d", len(results))
		}

		if results[0].Symbol != "ETHUSDT" || results[1].Symbol != "BTCUSDT" {
			t.Fatalf("orders not returned in expected order: %+v", results)
		}
	})

	t.Run("filters by user and exchange", func(t *testing.T) {
		mockRows := orderRows(orders[0])
		exchangeID := uint(1)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE user_id = $1 AND exchange_id = $2 ORDER BY created_at DESC, id DESC`)).
			WithArgs(uint(1), exchangeID).
			WillReturnRows(mockRows)

		results, err := repo.Search(context.Background(), OrderSearchOptions{UserID: 1, ExchangeID: &exchangeID})
		if err != nil {
			t.Fatalf("unexpected error searching orders: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("expected 1 order for user 1 and exchange 1, got %d", len(results))
		}

		if results[0].Symbol != "BTCUSDT" {
			t.Fatalf("unexpected order returned: %+v", results[0])
		}
	})

	t.Run("filters by symbol and created window", func(t *testing.T) {
		mockRows := orderRows(orders[1])
		filters := OrderSearchOptions{
			UserID:        1,
			Symbol:        ptrString("ETHUSDT"),
			CreatedAfter:  ptrTime(createdAt.Add(-time.Hour)),
			CreatedBefore: ptrTime(createdAt.Add(36 * time.Hour)),
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE user_id = $1 AND symbol = $2 AND created_at >= $3 AND created_at <= $4 ORDER BY created_at DESC, id DESC`)).
			WithArgs(uint(1), *filters.Symbol, *filters.CreatedAfter, *filters.CreatedBefore).
			WillReturnRows(mockRows)

		results, err := repo.Search(context.Background(), filters)
		if err != nil {
			t.Fatalf("unexpected error searching orders: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("expected 1 order for symbol filter, got %d", len(results))
		}

		if results[0].Symbol != "ETHUSDT" {
			t.Fatalf("unexpected order returned: %+v", results[0])
		}
	})

	t.Run("applies pagination", func(t *testing.T) {
		mockRows := orderRows(orders[0])
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "orders" WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`)).
			WithArgs(uint(1), 1, 1).
			WillReturnRows(mockRows)

		results, err := repo.Search(context.Background(), OrderSearchOptions{UserID: 1, Limit: 1, Offset: 1})
		if err != nil {
			t.Fatalf("unexpected error searching orders: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("expected 1 order for pagination, got %d", len(results))
		}

		if results[0].Symbol != "BTCUSDT" {
			t.Fatalf("unexpected paginated order: %+v", results[0])
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	})

	gdb, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		sqlDB.Close()
		t.Fatalf("failed to open gorm DB with sqlmock: %v", err)
	}

	return gdb, mock
}

func ptrString(val string) *string {
	return &val
}

func ptrTime(val time.Time) *time.Time {
	return &val
}
