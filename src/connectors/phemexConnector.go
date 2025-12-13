// FULL REST API CLIENT FOR PHEMEX USDT-M FUTURES
// RESTY ONLY + INTERNAL RETRY
package connectors

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	logger "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

// -----------------------------
// CONFIG
// -----------------------------
const (
	// Default retry configuration
	defaultRetryAttempts   = 5
	defaultRetryBaseDelay  = 500 * time.Millisecond
	defaultRetryMaxBackoff = 8 * time.Second
)

// -----------------------------
// API RESPONSE WRAPPER
// -----------------------------
type APIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// -----------------------------
// B) STRUCTURES FOR POSITIONS
// -----------------------------
type GAccountPositions struct {
	Account struct {
		UserID           int64  `json:"userID"`
		AccountID        int64  `json:"accountId"`
		Currency         string `json:"currency"`
		AccountBalanceRv string `json:"accountBalanceRv"`
	} `json:"account"`

	Positions []struct {
		AccountID        int64  `json:"accountID"`
		Symbol           string `json:"symbol"`
		Currency         string `json:"currency"`
		Side             string `json:"side"`
		PosSide          string `json:"posSide"`
		SizeRq           string `json:"sizeRq"`
		AvgEntryPriceRp  string `json:"avgEntryPriceRp"`
		PositionMarginRv string `json:"positionMarginRv"`
		MarkPriceRp      string `json:"markPriceRp"`
	} `json:"positions"`
}

// -----------------------------
// A) AUTHENTICATED CLIENT
// -----------------------------
type Client struct {
	apiKey    string
	apiSecret string
	baseURL   string
	http      *resty.Client
}

func isRetryableResp(r *resty.Response, err error) bool {
	if err != nil {
		return true
	}

	if r == nil {
		return false
	}

	code := r.StatusCode()

	if code >= 500 && code <= 599 {
		return true
	}
	if code == 429 {
		return true
	}
	if code == 408 {
		return true
	}
	return false
}

func NewClient(apiKey, apiSecret, baseURL string) *Client {
	retryCount := defaultRetryAttempts - 1

	if baseURL == "" {
		baseURL = "https://testnet-api.phemex.com"
		logger.Warn("No base URL provided, using default: %s", baseURL)
	}

	httpClient := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(15 * time.Second).
		SetRetryCount(retryCount).
		SetRetryWaitTime(defaultRetryBaseDelay).
		SetRetryMaxWaitTime(defaultRetryMaxBackoff).
		AddRetryCondition(isRetryableResp)

	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		http:      httpClient,
	}
}

func signRequest(path, query, body string, expiry int64, secret string) string {
	base := path
	if query != "" {
		base += query
	}
	base += fmt.Sprintf("%d", expiry)
	if body != "" {
		base += body
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) doRequest(method, path, query string, body []byte) (*APIResponse, error) {
	expiry := time.Now().Add(1 * time.Minute).Unix()

	sig := signRequest(path, query, string(body), expiry, c.apiSecret)

	req := c.http.R().
		SetHeader("x-phemex-access-token", c.apiKey).
		SetHeader("x-phemex-request-expiry", fmt.Sprintf("%d", expiry)).
		SetHeader("x-phemex-request-signature", sig)

	if query != "" {
		req = req.SetQueryString(query)
	}
	if body != nil {
		req = req.SetBody(body).SetHeader("Content-Type", "application/json")
	}

	resp, err := req.Execute(method, path)
	if err != nil {
		return nil, err
	}

	raw := resp.Body()

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode(), string(raw))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

// -----------------------------
// B) ACCOUNT & POSITION METHODS
// -----------------------------
func (c *Client) GetPositionsUSDT() (*GAccountPositions, error) {
	resp, err := c.doRequest("GET", "/g-accounts/positions", "currency=USDT", nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("API error: %s", resp.Msg)
	}

	var parsed GAccountPositions
	return &parsed, json.Unmarshal(resp.Data, &parsed)
}

// -----------------------------
// C) TRADING METHODS
// -----------------------------
func (c *Client) PlaceOrder(symbol, side, posSide, qty, ordType string, reduce bool) (*APIResponse, error) {
	body := map[string]interface{}{
		"symbol":      symbol,
		"side":        side,
		"posSide":     posSide,
		"ordType":     ordType,
		"orderQtyRq":  qty,
		"reduceOnly":  reduce,
		"clOrdID":     fmt.Sprintf("go-%d", time.Now().UnixNano()),
		"timeInForce": "ImmediateOrCancel",
	}

	b, _ := json.Marshal(body)
	return c.doRequest("POST", "/g-orders", "", b)
}

func (c *Client) CancelAll(symbol string) (*APIResponse, error) {
	return c.doRequest("DELETE", "/g-orders/all", fmt.Sprintf("symbol=%s", symbol), nil)
}

// -----------------------------
// D) ORDER QUERY METHODS
// -----------------------------
func (c *Client) GetActiveOrders(symbol string) (*APIResponse, error) {
	return c.doRequest("GET", "/g-orders/activeList", fmt.Sprintf("symbol=%s", symbol), nil)
}

func (c *Client) GetOrderHistory(symbol string) (*APIResponse, error) {
	return c.doRequest("GET", "/g-orders/trade/history", fmt.Sprintf("symbol=%s", symbol), nil)
}

func (c *Client) GetFills(symbol string) (*APIResponse, error) {
	return c.doRequest("GET", "/g-trades/fills", fmt.Sprintf("symbol=%s", symbol), nil)
}

// -----------------------------
// E) MARKET DATA METHODS
// -----------------------------
type mdResponse struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result json.RawMessage `json:"result"`
}

func (c *Client) GetTicker(symbol string) (*APIResponse, error) {
	resp, err := c.http.R().
		SetQueryParam("symbol", symbol).
		Get("/md/v3/ticker/24hr")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	var md mdResponse
	if err := json.Unmarshal(resp.Body(), &md); err != nil {
		return nil, err
	}
	if md.Error != nil {
		return nil, errors.New(md.Error.Message)
	}

	return &APIResponse{Code: 0, Data: md.Result}, nil
}

func (c *Client) GetOrderbook(symbol string) (*APIResponse, error) {
	resp, err := c.http.R().
		SetQueryParam("symbol", symbol).
		Get("/md/v2/orderbook")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	var md mdResponse
	if err := json.Unmarshal(resp.Body(), &md); err != nil {
		return nil, err
	}
	if md.Error != nil {
		return nil, errors.New(md.Error.Message)
	}

	return &APIResponse{Code: 0, Data: md.Result}, nil
}

func (c *Client) GetKlines(symbol string, res int) (*APIResponse, error) {
	return c.doRequest("GET", "/md/perpetual/kline",
		fmt.Sprintf("symbol=%s&resolution=%d", symbol, res),
		nil,
	)
}

// -----------------------------
// F) RISK & MARGIN
// -----------------------------
type RiskUnit struct {
	UserID                int64   `json:"userId"`
	RiskMode              string  `json:"riskMode"`
	ValuationCcy          int     `json:"valuationCcy"`
	Symbol                string  `json:"symbol"`
	PosSide               string  `json:"posSide"`
	TotalEquityRv         float64 `json:"totalEquityRv"`
	EstAvailableBalanceRv float64 `json:"estAvailableBalanceRv"`
	TotalPosCostRv        float64 `json:"totalPosCostRv"`
	TotalOrdUsedBalanceRv float64 `json:"totalOrdUsedBalanceRv"`
	FixedUsedRv           float64 `json:"fixedUsedRv"`
}

func (c *Client) GetFuturesAvailableFromRiskUnit(symbol string) (float64, error) {
	resp, err := c.doRequest("GET", "/g-accounts/risk-unit", "", nil)
	if err != nil {
		return 0, err
	}

	var units []RiskUnit
	if err := json.Unmarshal(resp.Data, &units); err != nil {
		return 0, err
	}

	for _, u := range units {
		if u.Symbol == symbol {
			if u.EstAvailableBalanceRv > 0 {
				return u.EstAvailableBalanceRv, nil
			}
			available := u.TotalEquityRv -
				u.TotalPosCostRv -
				u.TotalOrdUsedBalanceRv -
				u.FixedUsedRv

			if available < 0 {
				return 0, nil
			}
			return available, nil
		}
	}

	return 0, fmt.Errorf("no risk unit found for %s", symbol)
}

// -----------------------------
// G) USDT → BASE CONVERSION
// -----------------------------
func (c *Client) GetAvailableBaseFromUSDT(
	symbol string,
) (baseSymbol string, baseAvail float64, usdtAvail float64, price float64, err error) {

	if !strings.HasSuffix(symbol, "USDT") {
		err = fmt.Errorf("symbol must end in USDT: %s", symbol)
		return
	}

	baseSymbol = strings.TrimSuffix(symbol, "USDT")

	usdtAvail, err = c.GetFuturesAvailableFromRiskUnit(symbol)
	if err != nil {
		return
	}

	ticker, err := c.GetTicker(symbol)
	if err != nil {
		return
	}

	var tk struct {
		LastRp string `json:"lastRp"`
	}
	if err = json.Unmarshal(ticker.Data, &tk); err != nil {
		return
	}

	price, err = strconv.ParseFloat(tk.LastRp, 64)
	if err != nil || price <= 0 {
		err = fmt.Errorf("invalid price for %s", symbol)
		return
	}

	baseAvail = usdtAvail / price
	return
}

// CloseAllPositions closes ALL open positions (Long and Short) for a given symbol
// by sending MARKET orders in the opposite direction with reduceOnly enabled.
// This guarantees that positions are fully closed and no new positions are opened.
func (c *Client) CloseAllPositions(symbol string) error {
	logger.WithFields(map[string]interface{}{
		"symbol": symbol,
	}).Info("Closing ALL positions for symbol")

	// 1) Fetch all USDT positions from the account
	positions, err := c.GetPositionsUSDT()
	if err != nil {
		return fmt.Errorf("GetPositionsUSDT failed: %w", err)
	}

	// 2) Iterate through positions and filter by symbol
	for _, p := range positions.Positions {
		if p.Symbol != symbol {
			continue
		}

		// Skip empty positions (nothing to close)
		if p.SizeRq == "0" || p.SizeRq == "" {
			continue
		}

		// Determine the opposite side required to close the position
		var closeSide string
		switch p.Side {
		case "Buy":
			closeSide = "Sell"
		case "Sell":
			closeSide = "Buy"
		default:
			logger.WithFields(map[string]interface{}{
				"symbol": symbol,
				"side":   p.Side,
			}).Error("Unknown position side, skipping")
			continue
		}

		logger.WithFields(map[string]interface{}{
			"symbol":    p.Symbol,
			"posSide":   p.PosSide,
			"side":      p.Side,
			"size":      p.SizeRq,
			"closeSide": closeSide,
		}).Info("Closing position")

		// 3) Send a MARKET order with reduceOnly to fully close the position
		_, err := c.PlaceOrder(
			p.Symbol,  // trading pair
			closeSide, // opposite side to close the position
			p.PosSide, // Long or Short
			p.SizeRq,  // full position size
			"Market",  // market order
			true,      // reduceOnly = true (guarantees position close)
		)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"symbol":  p.Symbol,
				"posSide": p.PosSide,
				"side":    p.Side,
				"size":    p.SizeRq,
			}).WithError(err).Error("Failed to close position")

			return fmt.Errorf(
				"failed to close position %s %s (%s): %w",
				p.Symbol,
				p.PosSide,
				p.Side,
				err,
			)
		}
	}

	logger.WithFields(map[string]interface{}{
		"symbol": symbol,
	}).Info("All positions successfully closed")

	return nil
}
