package connectors

// Test index:
//  1. TestIsRetryableResp verifies retry decisions for various response codes and errors.
//  2. TestSignRequest validates HMAC signature generation inputs and output.
//  3. TestGetPositionsUSDT checks decoding of position data for USDT pairs.
//  4. TestTradingEndpoints ensures trading endpoints are called with expected methods and paths.
//  5. TestMarketDataEndpoints covers ticker and orderbook market data retrieval.
//  6. TestGetFuturesAvailableFromRiskUnit validates available balance retrieval for futures risk units.
//  7. TestGetFuturesAvailableFromRiskUnitMissingSymbol errors when symbol is absent from the response.
//  8. TestPhemexGetAvailableBaseFromUSDT_InvalidSymbol asserts rejection of non-USDT symbols.
//  9. TestPhemexGetAvailableBaseFromUSDT checks available base calculation from USDT balance and price.
// 10. TestPhemexGetAvailableBaseFromUSDTBadPrice validates errors for malformed ticker data.
// 11. TestGetKlines verifies the klines endpoint wiring.
// 12. TestCloseAllPositions ensures positions are closed by placing opposite orders.
// 13. TestCloseAllPositionsPlaceOrderError confirms errors propagate when closing orders fail.
// 14. TestCloseAllPositionsNoPositions verifies empty position lists exit without placing orders.
// 15. TestCloseAllPositionsUnknownSide returns an error when the position side is unknown.
// 16. TestPlaceStopLossOrder builds the stop loss request payload and wiring.
// 17. TestPlaceStopLossOrderValidation ensures required stop loss parameters are validated.
// 18. TestSetStopLossForOpenPosition walks the happy path for open-position stop loss placement.
// 19. TestSetStopLossForOpenPositionErrors surfaces missing positions and size zero errors.
// 20. TestSetStopLossForSymbolHedgeMode covers dual-side stop creation and validation errors.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

func newTestClient(baseURL string, httpClient *http.Client) *Client {
	restyClient := resty.New()
	restyClient.SetBaseURL(baseURL)
	restyClient.SetTransport(httpClient.Transport)

	return &Client{
		apiKey:    "test-key",
		apiSecret: "test-secret",
		baseURL:   baseURL,
		http:      restyClient,
	}
}

// TestIsRetryableResp verifies retry decisions for assorted errors and HTTP responses.
func TestIsRetryableResp(t *testing.T) {
	// Validates retry logic by exercising error presence and specific HTTP status codes to confirm
	// true is returned for retryable cases and false otherwise.
	cases := []struct {
		name string
		resp *resty.Response
		err  error
		want bool
	}{
		{name: "error present", err: assertError{}, want: true},
		{name: "server error", resp: fakeResponse(500), want: true},
		{name: "too many requests", resp: fakeResponse(429), want: true},
		{name: "timeout", resp: fakeResponse(408), want: true},
		{name: "ok response", resp: fakeResponse(200), want: false},
		{name: "nil resp", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isRetryableResp(tc.resp, tc.err)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

// TestSignRequest ensures HMAC signing matches the expected digest for a fixed payload and secret.
func TestSignRequest(t *testing.T) {
	// Ensures the HMAC signature matches the expected digest for a fixed request path, query,
	// body, and expiry using a known secret.
	expiry := int64(1700000000)
	expectedMac := hmac.New(sha256.New, []byte("secret"))
	expectedMac.Write([]byte("/testpath" + "query" + "1700000000" + "body"))
	expected := hex.EncodeToString(expectedMac.Sum(nil))

	got := signRequest("/testpath", "query", "body", expiry, "secret")
	if got != expected {
		t.Fatalf("expected signature %s, got %s", expected, got)
	}
}

// TestGetPositionsUSDT validates USDT position retrieval and decoding of the server payload.
func TestGetPositionsUSDT(t *testing.T) {
	// Confirms USDT position retrieval decodes the server payload and returns the expected
	// symbol details.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON(GAccountPositions{Positions: []struct {
			AccountID        int64  `json:"accountID"`
			Symbol           string `json:"symbol"`
			Currency         string `json:"currency"`
			Side             string `json:"side"`
			PosSide          string `json:"posSide"`
			SizeRq           string `json:"sizeRq"`
			AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
			PositionMarginRv string `json:"positionMarginRv"`
			MarkPriceRp      string `json:"markPriceRp"`
		}{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "2"}}})})
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	positions, err := client.GetPositionsUSDT()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(positions.Positions) != 1 || positions.Positions[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected positions: %+v", positions.Positions)
	}
}

// TestTradingEndpoints confirms trading endpoints use the correct HTTP methods and paths.
func TestTradingEndpoints(t *testing.T) {
	// Verifies trading endpoints use correct HTTP methods and URLs by recording server calls
	// across place order, cancel all, active orders, order history, and fills.
	type call struct {
		path   string
		method string
	}
	var calls []call

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, call{path: r.URL.Path + "?" + r.URL.RawQuery, method: r.Method})
		switch r.URL.Path {
		case "/g-orders":
			_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON(map[string]string{"orderID": "1"})})
		case "/g-orders/all", "/g-orders/activeList", "/g-orders/trade/history", "/g-trades/fills":
			_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON(map[string]string{"ok": "true"})})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())

	if _, err := client.PlaceOrder("BTCUSDT", "Buy", "Long", "1", "Market", false); err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}
	if _, err := client.CancelAll("BTCUSDT"); err != nil {
		t.Fatalf("CancelAll error: %v", err)
	}
	if _, err := client.GetActiveOrders("BTCUSDT"); err != nil {
		t.Fatalf("GetActiveOrders error: %v", err)
	}
	if _, err := client.GetOrderHistory("BTCUSDT"); err != nil {
		t.Fatalf("GetOrderHistory error: %v", err)
	}
	if _, err := client.GetFills("BTCUSDT"); err != nil {
		t.Fatalf("GetFills error: %v", err)
	}

	expected := []call{
		{path: "/g-orders?", method: http.MethodPost},
		{path: "/g-orders/all?symbol=BTCUSDT", method: http.MethodDelete},
		{path: "/g-orders/activeList?symbol=BTCUSDT", method: http.MethodGet},
		{path: "/g-orders/trade/history?symbol=BTCUSDT", method: http.MethodGet},
		{path: "/g-trades/fills?symbol=BTCUSDT", method: http.MethodGet},
	}

	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d", len(expected), len(calls))
	}
	for i, e := range expected {
		if calls[i] != e {
			t.Fatalf("call %d expected %+v got %+v", i, e, calls[i])
		}
	}
}

// TestMarketDataEndpoints checks ticker and orderbook retrieval and response parsing.
func TestMarketDataEndpoints(t *testing.T) {
	// Checks market data endpoints for ticker and orderbook to ensure responses are parsed and
	// returned as expected.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/md/v3/ticker/24hr":
			_ = json.NewEncoder(w).Encode(mdResponse{Result: []byte(`{"lastRp":"60000"}`)})
		case "/md/v2/orderbook":
			_ = json.NewEncoder(w).Encode(mdResponse{Result: []byte(`{"book":"ok"}`)})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())

	ticker, err := client.GetTicker("BTCUSDT")
	if err != nil {
		t.Fatalf("GetTicker error: %v", err)
	}
	if string(ticker.Data) != `{"lastRp":"60000"}` {
		t.Fatalf("unexpected ticker data: %s", string(ticker.Data))
	}

	ob, err := client.GetOrderbook("BTCUSDT")
	if err != nil {
		t.Fatalf("GetOrderbook error: %v", err)
	}
	if string(ob.Data) != `{"book":"ok"}` {
		t.Fatalf("unexpected orderbook data: %s", string(ob.Data))
	}
}

// TestGetFuturesAvailableFromRiskUnit validates available balance retrieval from the risk unit endpoint.
func TestGetFuturesAvailableFromRiskUnit(t *testing.T) {
	// Validates available balance retrieval from the risk unit endpoint and ensures errors are
	// raised when the requested symbol is missing.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON([]RiskUnit{{
			Symbol:                "BTCUSDT",
			EstAvailableBalanceRv: 50,
		}})})
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	v, err := client.GetFuturesAvailableFromRiskUnit("BTCUSDT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 50 {
		t.Fatalf("expected 50, got %f", v)
	}

	if _, err := client.GetFuturesAvailableFromRiskUnit("ETHUSDT"); err == nil {
		t.Fatalf("expected error for missing symbol")
	}
}

// TestGetFuturesAvailableFromRiskUnitMissingSymbol captures the missing symbol branch explicitly.
func TestGetFuturesAvailableFromRiskUnitMissingSymbol(t *testing.T) {
	// Ensures the helper reports an error when the requested symbol is not contained in the
	// risk unit response payload.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON([]RiskUnit{{
			Symbol:                "ETHUSDT",
			EstAvailableBalanceRv: 10,
		}})})
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	if _, err := client.GetFuturesAvailableFromRiskUnit("BTCUSDT"); err == nil {
		t.Fatalf("expected error for missing BTCUSDT risk unit")
	}
}

// TestPhemexGetAvailableBaseFromUSDT_InvalidSymbol rejects non-USDT symbols before remote calls.
func TestPhemexGetAvailableBaseFromUSDT_InvalidSymbol(t *testing.T) {
	// Ensures non-USDT symbols are rejected and produce an error before any remote calls.
	client := newTestClient("http://example", resty.New().GetClient())
	if _, _, _, _, err := client.GetAvailableBaseFromUSDT("BTCUSD"); err == nil {
		t.Fatalf("expected error for non-USDT symbol")
	}
}

// TestPhemexGetAvailableBaseFromUSDT calculates base availability using USDT balance and ticker price data.
func TestPhemexGetAvailableBaseFromUSDT(t *testing.T) {
	// Confirms base currency availability is calculated from USDT balance and ticker price and
	// validates the parsed base symbol, available USDT, and derived base amount.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/risk-unit":
			resp := APIResponse{Code: 0, Data: mustJSON([]RiskUnit{{Symbol: "BTCUSDT", EstAvailableBalanceRv: 1000}})}
			_ = json.NewEncoder(w).Encode(resp)
		case "/md/v3/ticker/24hr":
			_ = json.NewEncoder(w).Encode(mdResponse{Result: []byte(`{"lastRp":"50000"}`)})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())

	base, baseAvail, usdtAvail, price, err := client.GetAvailableBaseFromUSDT("BTCUSDT")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if base != "BTC" {
		t.Fatalf("expected base BTC, got %s", base)
	}

	if usdtAvail != 1000 {
		t.Fatalf("expected usdt available 1000, got %f", usdtAvail)
	}

	if price != 50000 {
		t.Fatalf("expected price 50000, got %f", price)
	}

	expectedBase := usdtAvail / price
	if baseAvail != expectedBase {
		t.Fatalf("expected base available %f, got %f", expectedBase, baseAvail)
	}
}

// TestPhemexGetAvailableBaseFromUSDTBadPrice ensures malformed ticker prices are rejected.
func TestPhemexGetAvailableBaseFromUSDTBadPrice(t *testing.T) {
	// Validates that invalid ticker prices propagate an error so callers do not compute an
	// available base amount from malformed data.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/risk-unit":
			_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON([]RiskUnit{{Symbol: "BTCUSDT", EstAvailableBalanceRv: 1000}})})
		case "/md/v3/ticker/24hr":
			_ = json.NewEncoder(w).Encode(mdResponse{Result: []byte(`{"lastRp":"bad"}`)})
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	if _, _, _, _, err := client.GetAvailableBaseFromUSDT("BTCUSDT"); err == nil {
		t.Fatalf("expected error when ticker price is invalid")
	}
}

// TestGetKlines confirms the klines endpoint is invoked correctly.
func TestGetKlines(t *testing.T) {
	// Ensures the klines helper points at the documented endpoint with the provided symbol and
	// resolution parameters.
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path + "?" + r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(APIResponse{Code: 0})
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	if _, err := client.GetKlines("BTCUSDT", 5); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if path != "/md/perpetual/kline?resolution=5&symbol=BTCUSDT" {
		t.Fatalf("unexpected klines path: %s", path)
	}
}

// TestCloseAllPositions ensures closing orders are issued for existing positions.
func TestCloseAllPositions(t *testing.T) {
	// Ensures existing positions trigger a closing market order and tracks the number of
	// generated orders to confirm all positions are addressed.
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/positions":
			resp := APIResponse{Code: 0, Data: mustJSON(GAccountPositions{Positions: []struct {
				AccountID        int64  `json:"accountID"`
				Symbol           string `json:"symbol"`
				Currency         string `json:"currency"`
				Side             string `json:"side"`
				PosSide          string `json:"posSide"`
				SizeRq           string `json:"sizeRq"`
				AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
				PositionMarginRv string `json:"positionMarginRv"`
				MarkPriceRp      string `json:"markPriceRp"`
			}{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}})}
			_ = json.NewEncoder(w).Encode(resp)
		case "/g-orders":
			callCount++
			resp := APIResponse{Code: 0, Data: mustJSON(map[string]string{"orderID": "1"})}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())

	if err := client.CloseAllPositions("BTCUSDT"); err != nil {
		t.Fatalf("expected no error closing positions, got %v", err)
	}

	if callCount != 1 {
		t.Fatalf("expected one closing order to be placed, got %d", callCount)
	}
}

// TestCloseAllPositionsPlaceOrderError propagates errors when closing orders fail to place.
func TestCloseAllPositionsPlaceOrderError(t *testing.T) {
	// Confirms that an error is returned when placing a closing order fails after fetching
	// positions to close.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/positions":
			resp := APIResponse{Code: 0, Data: mustJSON(GAccountPositions{Positions: []struct {
				AccountID        int64  `json:"accountID"`
				Symbol           string `json:"symbol"`
				Currency         string `json:"currency"`
				Side             string `json:"side"`
				PosSide          string `json:"posSide"`
				SizeRq           string `json:"sizeRq"`
				AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
				PositionMarginRv string `json:"positionMarginRv"`
				MarkPriceRp      string `json:"markPriceRp"`
			}{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"}}})}
			_ = json.NewEncoder(w).Encode(resp)
		case "/g-orders":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())

	if err := client.CloseAllPositions("BTCUSDT"); err == nil {
		t.Fatalf("expected error when place order fails")
	}
}

// TestCloseAllPositionsNoPositions verifies empty position lists return without placing orders.
func TestCloseAllPositionsNoPositions(t *testing.T) {
	// Ensures a call with no matching positions short-circuits without hitting the order
	// endpoint and still returns a nil error.
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/positions":
			emptyPositions := GAccountPositions{Positions: []struct {
				AccountID        int64  `json:"accountID"`
				Symbol           string `json:"symbol"`
				Currency         string `json:"currency"`
				Side             string `json:"side"`
				PosSide          string `json:"posSide"`
				SizeRq           string `json:"sizeRq"`
				AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
				PositionMarginRv string `json:"positionMarginRv"`
				MarkPriceRp      string `json:"markPriceRp"`
			}{}}
			resp := APIResponse{Code: 0, Data: mustJSON(emptyPositions)}
			_ = json.NewEncoder(w).Encode(resp)
		case "/g-orders":
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())

	if err := client.CloseAllPositions("BTCUSDT"); err != nil {
		t.Fatalf("expected no error closing positions, got %v", err)
	}

	if callCount != 0 {
		t.Fatalf("expected no orders to be placed, got %d", callCount)
	}
}

// TestCloseAllPositionsUnknownSide reports errors when the position side is not recognized.
func TestCloseAllPositionsUnknownSide(t *testing.T) {
	// Ensures a position with an unknown side surfaces an error instead of silently skipping
	// the invalid record.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/positions":
			resp := APIResponse{Code: 0, Data: mustJSON(GAccountPositions{Positions: []struct {
				AccountID        int64  `json:"accountID"`
				Symbol           string `json:"symbol"`
				Currency         string `json:"currency"`
				Side             string `json:"side"`
				PosSide          string `json:"posSide"`
				SizeRq           string `json:"sizeRq"`
				AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
				PositionMarginRv string `json:"positionMarginRv"`
				MarkPriceRp      string `json:"markPriceRp"`
			}{{Symbol: "BTCUSDT", Side: "Unknown", PosSide: "Long", SizeRq: "1"}}})}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	if err := client.CloseAllPositions("BTCUSDT"); err == nil {
		t.Fatalf("expected error for unknown position side")
	}
}

// TestPlaceStopLossOrder validates the stop loss request payload and wiring.
func TestPlaceStopLossOrder(t *testing.T) {
	// Confirms stop loss requests are issued to the correct endpoint with the expected fields
	// for hedged positions.
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON(map[string]string{"ok": "true"})})
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	if _, err := client.PlaceStopLossOrder("BTCUSDT", "Long", "Sell", "2", "30000", TriggerByMarkPrice, true); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if captured["symbol"] != "BTCUSDT" || captured["posSide"] != "Long" || captured["side"] != "Sell" || captured["ordType"] != "Stop" {
		t.Fatalf("unexpected stop loss payload: %+v", captured)
	}
	if captured["reduceOnly"] != true || captured["closeOnTrigger"] != true {
		t.Fatalf("expected reduceOnly and closeOnTrigger to be true, got %+v", captured)
	}
}

// TestPlaceStopLossOrderValidation enforces required arguments.
func TestPlaceStopLossOrderValidation(t *testing.T) {
	// Ensures errors are returned when required stop loss parameters are empty.
	client := newTestClient("http://example", resty.New().GetClient())
	if _, err := client.PlaceStopLossOrder("", "Long", "Sell", "1", "30000", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected validation error for empty symbol")
	}
	if _, err := client.PlaceStopLossOrder("BTCUSDT", "", "Sell", "1", "30000", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected validation error for empty posSide")
	}
	if _, err := client.PlaceStopLossOrder("BTCUSDT", "Long", "", "1", "30000", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected validation error for empty side")
	}
	if _, err := client.PlaceStopLossOrder("BTCUSDT", "Long", "Sell", "", "30000", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected validation error for empty qty")
	}
	if _, err := client.PlaceStopLossOrder("BTCUSDT", "Long", "Sell", "1", "", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected validation error for empty stop price")
	}
}

// TestSetStopLossForOpenPosition checks stop placement using the open position size.
func TestSetStopLossForOpenPosition(t *testing.T) {
	// Confirms the helper identifies the open position side, derives the opposite order side,
	// and delegates to PlaceStopLossOrder with the position size.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/positions":
			resp := APIResponse{Code: 0, Data: mustJSON(GAccountPositions{Positions: []struct {
				AccountID        int64  `json:"accountID"`
				Symbol           string `json:"symbol"`
				Currency         string `json:"currency"`
				Side             string `json:"side"`
				PosSide          string `json:"posSide"`
				SizeRq           string `json:"sizeRq"`
				AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
				PositionMarginRv string `json:"positionMarginRv"`
				MarkPriceRp      string `json:"markPriceRp"`
			}{{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "3"}}})}
			_ = json.NewEncoder(w).Encode(resp)
		case "/g-orders":
			_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON(map[string]string{"orderID": "sl"})})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	if _, err := client.SetStopLossForOpenPosition("BTCUSDT", "Long", "30000", TriggerByMarkPrice, true); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// TestSetStopLossForOpenPositionErrors checks missing positions and zero sizes.
func TestSetStopLossForOpenPositionErrors(t *testing.T) {
	// Ensures missing positions, zero sizes, and unknown sides all return errors before placing
	// stop orders.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/positions":
			resp := APIResponse{Code: 0, Data: mustJSON(GAccountPositions{Positions: []struct {
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
				{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "0"},
				{Symbol: "BTCUSDT", Side: "Unknown", PosSide: "Short", SizeRq: "1"},
			}})}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			_ = json.NewEncoder(w).Encode(APIResponse{Code: 0})
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	if _, err := client.SetStopLossForOpenPosition("BTCUSDT", "Long", "30000", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected error for zero-sized position")
	}
	if _, err := client.SetStopLossForOpenPosition("BTCUSDT", "Short", "30000", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected error for unknown side")
	}
	if _, err := client.SetStopLossForOpenPosition("ETHUSDT", "Long", "30000", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected error when position not found")
	}
}

// TestSetStopLossForSymbolHedgeMode covers dual-side stop placement and validation errors.
func TestSetStopLossForSymbolHedgeMode(t *testing.T) {
	// Ensures both long and short stop losses are placed when provided, and errors bubble when
	// neither price is supplied or inner calls fail.
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g-accounts/positions":
			resp := APIResponse{Code: 0, Data: mustJSON(GAccountPositions{Positions: []struct {
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
				{Symbol: "BTCUSDT", Side: "Buy", PosSide: "Long", SizeRq: "1"},
				{Symbol: "BTCUSDT", Side: "Sell", PosSide: "Short", SizeRq: "2"},
			}})}
			_ = json.NewEncoder(w).Encode(resp)
		case "/g-orders":
			calls++
			_ = json.NewEncoder(w).Encode(APIResponse{Code: 0, Data: mustJSON(map[string]string{"orderID": fmt.Sprintf("sl-%d", calls)})})
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, server.Client())
	res, err := client.SetStopLossForSymbolHedgeMode("BTCUSDT", "30000", "31000", TriggerByMarkPrice, true)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(res) != 2 || calls != 2 {
		t.Fatalf("expected two stop loss calls, got responses=%d calls=%d", len(res), calls)
	}

	if _, err := client.SetStopLossForSymbolHedgeMode("BTCUSDT", "", "", TriggerByMarkPrice, true); err == nil {
		t.Fatalf("expected error when no stop prices provided")
	}
}

type assertError struct{}

func (assertError) Error() string { return "err" }

func fakeResponse(status int) *resty.Response {
	return &resty.Response{RawResponse: &http.Response{StatusCode: status}}
}

func mustJSON(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
