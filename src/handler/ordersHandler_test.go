package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"adminapi/src/auth"
	"adminapi/src/model"
	"adminapi/src/repository"

	"github.com/stretchr/testify/assert"
)

type mockOrderSearcher struct {
	orders        []model.Order
	err           error
	userID        uint
	exchangeID    *uint
	symbol        *string
	createdAfter  *time.Time
	createdBefore *time.Time
	limit         int
	offset        int
	calledCount   int
}

func (m *mockOrderSearcher) Search(ctx context.Context, options repository.OrderSearchOptions) ([]model.Order, error) {
	m.calledCount++
	m.userID = options.UserID
	m.exchangeID = options.ExchangeID
	m.symbol = options.Symbol
	m.createdAfter = options.CreatedAfter
	m.createdBefore = options.CreatedBefore
	m.limit = options.Limit
	m.offset = options.Offset
	return m.orders, m.err
}

func TestSearchOrdersHandler_Unauthorized(t *testing.T) {
	handler := SearchOrdersHandler(&mockOrderSearcher{})

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestSearchOrdersHandler_InvalidExchange(t *testing.T) {
	handler := SearchOrdersHandler(&mockOrderSearcher{})

	req := httptest.NewRequest(http.MethodGet, "/orders?exchangeId=abc", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserKey, &model.User{ID: 1}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSearchOrdersHandler_RepoError(t *testing.T) {
	mockRepo := &mockOrderSearcher{err: assert.AnError}
	handler := SearchOrdersHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserKey, &model.User{ID: 42}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}

	if mockRepo.calledCount != 1 {
		t.Fatalf("expected repository to be called once, got %d", mockRepo.calledCount)
	}
}

func TestSearchOrdersHandler_Success(t *testing.T) {
	exchangeID := uint(2)
	orders := []model.Order{{ID: 1, Symbol: "BTCUSDT"}}
	mockRepo := &mockOrderSearcher{orders: orders}
	handler := SearchOrdersHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/orders?exchangeId=2&symbol=BTCUSDT&createdFrom=2024-01-01T00:00:00Z&createdTo=2024-02-01T00:00:00Z&page=2&pageSize=5", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserKey, &model.User{ID: 7}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if mockRepo.calledCount != 1 {
		t.Fatalf("expected repository to be called once, got %d", mockRepo.calledCount)
	}

	if mockRepo.userID != 7 {
		t.Fatalf("expected user ID 7, got %d", mockRepo.userID)
	}

	if mockRepo.exchangeID == nil || *mockRepo.exchangeID != exchangeID {
		t.Fatalf("expected exchange ID %d, got %v", exchangeID, mockRepo.exchangeID)
	}

	if mockRepo.symbol == nil || *mockRepo.symbol != "BTCUSDT" {
		t.Fatalf("expected symbol BTCUSDT, got %v", mockRepo.symbol)
	}

	if mockRepo.createdAfter == nil || mockRepo.createdBefore == nil {
		t.Fatalf("expected createdAt filters to be set")
	}

	if mockRepo.limit != 5 || mockRepo.offset != 5 {
		t.Fatalf("expected limit 5 and offset 5, got limit=%d offset=%d", mockRepo.limit, mockRepo.offset)
	}

	if rr.Body.String() == "" {
		t.Fatalf("expected response body to be set")
	}
}

func TestSearchOrdersHandler_InvalidPagination(t *testing.T) {
	handler := SearchOrdersHandler(&mockOrderSearcher{})

	req := httptest.NewRequest(http.MethodGet, "/orders?page=0", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserKey, &model.User{ID: 1}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSearchOrdersHandler_InvalidDate(t *testing.T) {
	handler := SearchOrdersHandler(&mockOrderSearcher{})

	req := httptest.NewRequest(http.MethodGet, "/orders?createdFrom=invalid", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserKey, &model.User{ID: 1}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
