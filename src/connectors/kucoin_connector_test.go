package connectors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockResponse builds a KuCoin-style response body.
func mockResponse(code string, data string) string {
	return `{"code":"` + code + `","data":` + data + `}`
}

func TestGetAvailableBaseFromUSDT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/account-overview":
			_, _ = w.Write([]byte(mockResponse("200000", `{"availableBalance":100,"currency":"USDT"}`)))
		case "/api/v1/ticker":
			_, _ = w.Write([]byte(mockResponse("200000", `{"price":"25000","lastTradePrice":"24900"}`)))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	connector := &KucoinConnector{
		spotClient: &kucoinRESTClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
		},
		futuresClient: &kucoinRESTClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
		},
	}

	baseSymbol, baseAvail, usdtAvail, price, err := connector.GetAvailableBaseFromUSDT("XBTUSDTM")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if baseSymbol != "XBT" {
		t.Fatalf("expected base symbol XBT, got %s", baseSymbol)
	}

	if usdtAvail != 100 {
		t.Fatalf("expected USDT available 100, got %f", usdtAvail)
	}

	if price != 25000 {
		t.Fatalf("expected ticker price 25000, got %f", price)
	}

	expectedBase := usdtAvail / price
	if baseAvail != expectedBase {
		t.Fatalf("expected base available %f, got %f", expectedBase, baseAvail)
	}
}
