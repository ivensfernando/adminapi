package connectors

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestIsRetryableResp(t *testing.T) {
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

func TestSignRequest(t *testing.T) {
	expiry := int64(1700000000)
	expectedMac := hmac.New(sha256.New, []byte("secret"))
	expectedMac.Write([]byte("/testpath" + "query" + "1700000000" + "body"))
	expected := hex.EncodeToString(expectedMac.Sum(nil))

	got := signRequest("/testpath", "query", "body", expiry, "secret")
	if got != expected {
		t.Fatalf("expected signature %s, got %s", expected, got)
	}
}

func TestGetPositionsUSDT(t *testing.T) {
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

func TestTradingEndpoints(t *testing.T) {
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

func TestMarketDataEndpoints(t *testing.T) {
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

func TestGetFuturesAvailableFromRiskUnit(t *testing.T) {
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

func TestPhemexGetAvailableBaseFromUSDT_InvalidSymbol(t *testing.T) {
	client := newTestClient("http://example", resty.New().GetClient())
	if _, _, _, _, err := client.GetAvailableBaseFromUSDT("BTCUSD"); err == nil {
		t.Fatalf("expected error for non-USDT symbol")
	}
}

func TestPhemexGetAvailableBaseFromUSDT(t *testing.T) {
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

func TestCloseAllPositions(t *testing.T) {
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

func TestCloseAllPositionsPlaceOrderError(t *testing.T) {
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

type assertError struct{}

func (assertError) Error() string { return "err" }

func fakeResponse(status int) *resty.Response {
	return &resty.Response{RawResponse: &http.Response{StatusCode: status}}
}

func mustJSON(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
