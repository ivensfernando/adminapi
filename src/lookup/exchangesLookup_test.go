package lookup

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"adminapi/src/model"
)

func TestListExchanges(t *testing.T) {
	originalFetcher := fetchExchanges
	t.Cleanup(func() { fetchExchanges = originalFetcher })

	fetchExchanges = func() ([]model.Exchange, int64, error) {
		return []model.Exchange{{Name: "Binance"}, {Name: "Kraken"}}, 2, nil
	}

	logger := logrus.NewEntry(logrus.New())
	req := httptest.NewRequest(http.MethodGet, "/exchanges", nil)
	rec := httptest.NewRecorder()

	ListExchanges(logger).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("X-Total-Count"); got != "2" {
		t.Fatalf("expected X-Total-Count to be 2, got %s", got)
	}

	var exchanges []model.Exchange
	if err := json.NewDecoder(rec.Body).Decode(&exchanges); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(exchanges) != 2 {
		t.Fatalf("expected 2 exchanges, got %d", len(exchanges))
	}

	if exchanges[0].Name != "Binance" || exchanges[1].Name != "Kraken" {
		t.Fatalf("unexpected exchanges returned: %+v", exchanges)
	}
}

func TestListExchanges_Error(t *testing.T) {
	originalFetcher := fetchExchanges
	t.Cleanup(func() { fetchExchanges = originalFetcher })

	fetchExchanges = func() ([]model.Exchange, int64, error) {
		return nil, 0, errors.New("boom")
	}

	logger := logrus.NewEntry(logrus.New())
	req := httptest.NewRequest(http.MethodGet, "/exchanges", nil)
	rec := httptest.NewRecorder()

	ListExchanges(logger).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestListPairs(t *testing.T) {
	originalFetcher := fetchPairs
	t.Cleanup(func() { fetchPairs = originalFetcher })

	fetchPairs = func() ([]model.PairsCoins, int64, error) {
		return []model.PairsCoins{{Display: "BTC/USDT"}, {Display: "ETH/USDT"}}, 2, nil
	}

	logger := logrus.NewEntry(logrus.New())
	req := httptest.NewRequest(http.MethodGet, "/pairs", nil)
	rec := httptest.NewRecorder()

	ListPairs(logger).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("X-Total-Count"); got != "2" {
		t.Fatalf("expected X-Total-Count to be 2, got %s", got)
	}

	var pairs []model.PairsCoins
	if err := json.NewDecoder(rec.Body).Decode(&pairs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}

	if pairs[0].Display != "BTC/USDT" || pairs[1].Display != "ETH/USDT" {
		t.Fatalf("unexpected pairs returned: %+v", pairs)
	}
}

func TestListPairs_Error(t *testing.T) {
	originalFetcher := fetchPairs
	t.Cleanup(func() { fetchPairs = originalFetcher })

	fetchPairs = func() ([]model.PairsCoins, int64, error) {
		return nil, 0, errors.New("boom")
	}

	logger := logrus.NewEntry(logrus.New())
	req := httptest.NewRequest(http.MethodGet, "/pairs", nil)
	rec := httptest.NewRecorder()

	ListPairs(logger).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
