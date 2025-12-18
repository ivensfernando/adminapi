package connectors

// FULL REST API CLIENT FOR KRAKEN FUTURES (v3 /derivatives)
// RESTY ONLY + INTERNAL RETRY

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	logger "github.com/sirupsen/logrus"
)

//// -----------------------------
//// CONFIG
//// -----------------------------
//const (
//	defaultRetryAttempts   = 5
//	defaultRetryBaseDelay  = 500 * time.Millisecond
//	defaultRetryMaxBackoff = 8 * time.Second
//)

// Kraken Futures uses /derivatives + /api/v3/...
const (
	defaultKrakenDerivativesBaseURL = "https://futures.kraken.com/derivatives"
	apiV3Prefix                     = "/api/v3"
)

type KrakenOrderbookResponse struct {
	Result     string `json:"result"`
	ServerTime string `json:"serverTime"`
	OrderBook  struct {
		Asks [][]float64 `json:"asks"`
		Bids [][]float64 `json:"bids"`
	} `json:"orderBook"`
	Error  string   `json:"error,omitempty"`
	Errors []string `json:"errors,omitempty"`
}

type KrakenOpenOrdersResponse struct {
	Result     string            `json:"result"`
	ServerTime string            `json:"serverTime"`
	OpenOrders []KrakenOpenOrder `json:"openOrders"`
	Error      string            `json:"error,omitempty"`
	Errors     []string          `json:"errors,omitempty"`
}

type KrakenOpenOrder struct {
	OrderID        string   `json:"order_id"`
	Symbol         string   `json:"symbol"`
	Side           string   `json:"side"`
	OrderType      string   `json:"orderType"`
	LimitPrice     *float64 `json:"limitPrice,omitempty"`
	ReduceOnly     bool     `json:"reduceOnly"`
	FilledSize     float64  `json:"filledSize"`
	UnfilledSize   float64  `json:"unfilledSize"`
	Status         string   `json:"status"`
	ReceivedTime   string   `json:"receivedTime"`
	LastUpdateTime string   `json:"lastUpdateTime"`
}

// -----------------------------
// CLIENT
// -----------------------------
type KrakenFuturesClient struct {
	apiKey    string
	apiSecret string // base64-encoded secret from Kraken
	baseURL   string
	http      *resty.Client
}

func NewKrakenFuturesClient(apiKey, apiSecret, baseURL string) *KrakenFuturesClient {
	retryCount := defaultRetryAttempts - 1

	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultKrakenDerivativesBaseURL
		logger.Warnf("No base URL provided, using default: %s", baseURL)
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(15 * time.Second).
		SetRetryCount(retryCount).
		SetRetryWaitTime(defaultRetryBaseDelay).
		SetRetryMaxWaitTime(defaultRetryMaxBackoff).
		AddRetryCondition(isRetryableResp)

	return &KrakenFuturesClient{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		http:      httpClient,
	}
}

// -----------------------------
// AUTH
// -----------------------------
//
// Kraken Futures REST (v3 /derivatives/*) Authent:
//  1) message = postData + Nonce + endpointPath
//  2) sha256(message)
//  3) base64-decode apiSecret
//  4) hmac-sha512(secretDecoded, sha256Digest)
//  5) base64-encode result
//
// endpointPath example in docs: /api/v3/orderbook (note. no /derivatives prefix) :contentReference[oaicite:1]{index=1}
//
// Important encoding note: Kraken is moving toward hashing the full url-encoded URI component "as sent". :contentReference[oaicite:2]{index=2}

func nonceMillis() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

func computeAuthent(postData, nonce, endpointPath, apiSecretB64 string) (string, error) {
	msg := postData + nonce + endpointPath

	sum := sha256.Sum256([]byte(msg))

	secret, err := base64.StdEncoding.DecodeString(apiSecretB64)
	if err != nil {
		return "", fmt.Errorf("base64 decode api secret failed: %w", err)
	}

	mac := hmac.New(sha512.New, secret)
	_, _ = mac.Write(sum[:])

	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// We sign exactly what we send. This helper encodes spaces as %20, not '+'.
// That makes the signed query closer to a raw URI component. :contentReference[oaicite:3]{index=3}
func queryEscapeRFC3986(s string) string {
	esc := url.QueryEscape(s)
	return strings.ReplaceAll(esc, "+", "%20")
}

func encodeValuesRFC3986(v url.Values) string {
	if v == nil || len(v) == 0 {
		return ""
	}
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := v[k]
		sort.Strings(vals)
		ek := queryEscapeRFC3986(k)
		for _, val := range vals {
			parts = append(parts, ek+"="+queryEscapeRFC3986(val))
		}
	}
	return strings.Join(parts, "&")
}

// -----------------------------
// LOW-LEVEL REQUESTS
// -----------------------------
type krakenBaseResp struct {
	Result     string `json:"result"`
	Error      string `json:"error,omitempty"`
	ServerTime string `json:"serverTime,omitempty"`
}

func (c *KrakenFuturesClient) doPublicRequest(method, endpoint string, params url.Values, out any) error {
	return c.doRequest(method, endpoint, params, false, out)
}

func (c *KrakenFuturesClient) doPrivateRequest(method, endpoint string, params url.Values, out any) error {
	return c.doRequest(method, endpoint, params, true, out)
}

func (c *KrakenFuturesClient) doRequest(method, endpoint string, params url.Values, auth bool, out any) error {
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// Actual HTTP path includes /api/v3 prefix under /derivatives.
	httpPath := apiV3Prefix + endpoint

	// Signature endpointPath is /api/v3/... (no /derivatives prefix). :contentReference[oaicite:4]{index=4}
	endpointPathForSig := apiV3Prefix + endpoint

	postData := encodeValuesRFC3986(params)

	req := c.http.R().
		SetHeader("Accept", "application/json")

	if auth {
		nonce := nonceMillis()
		authent, err := computeAuthent(postData, nonce, endpointPathForSig, c.apiSecret)
		if err != nil {
			return err
		}

		req = req.
			SetHeader("APIKey", c.apiKey).
			SetHeader("Nonce", nonce).
			SetHeader("Authent", authent)
	}

	// Kraken v3 endpoints accept parameters in the URL for many endpoints (including sendorder in docs).
	// We set the raw query string so signing matches what is sent.
	if postData != "" {
		req = req.SetQueryString(postData)
	}

	resp, err := req.Execute(method, httpPath)
	if err != nil {
		return err
	}

	raw := resp.Body()
	if resp.StatusCode() != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode(), string(raw))
	}

	// Many Kraken Futures endpoints return HTTP 200 even on errors, with {result:"error", error:"..."}.
	var base krakenBaseResp
	if err := json.Unmarshal(raw, &base); err != nil {
		return fmt.Errorf("json unmarshal failed: %w. raw=%s", err, string(raw))
	}
	if strings.EqualFold(base.Result, "error") {
		if base.Error == "" {
			return errors.New("kraken futures returned result=error")
		}
		return fmt.Errorf("kraken futures error: %s", base.Error)
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("json unmarshal into output failed: %w. raw=%s", err, string(raw))
		}
	}

	return nil
}

// -----------------------------
// TRADING
// -----------------------------
type SendOrderRequest struct {
	OrderType string  // required: lmt, post, ioc, mkt, stp, take_profit, trailing_stop, fok
	Symbol    string  // required: e.g. PF_XBTUSD
	Side      string  // required: buy, sell
	Size      float64 // required

	LimitPrice *float64 // optional
	StopPrice  *float64 // optional

	CliOrdID      *string // optional (<=100 chars)
	TriggerSignal *string // optional: mark, index, last
	ReduceOnly    *bool   // optional
	ProcessBefore *string // optional RFC3339Nano

	// Trailing stop extras (optional)
	TrailingStopMaxDeviation  *float64
	TrailingStopDeviationUnit *string // PERCENT or QUOTE_CURRENCY

	// Trigger offset (optional)
	LimitPriceOffsetValue *float64
	LimitPriceOffsetUnit  *string // QUOTE_CURRENCY or PERCENT

	Broker *string // optional (demo only)
}

func (r SendOrderRequest) toValues() (url.Values, error) {
	v := url.Values{}

	if strings.TrimSpace(r.OrderType) == "" {
		return nil, errors.New("orderType is required")
	}
	if strings.TrimSpace(r.Symbol) == "" {
		return nil, errors.New("symbol is required")
	}
	if strings.TrimSpace(r.Side) == "" {
		return nil, errors.New("side is required")
	}
	if r.Size <= 0 {
		return nil, errors.New("size must be > 0")
	}

	v.Set("orderType", strings.ToLower(r.OrderType))
	v.Set("symbol", r.Symbol)
	v.Set("side", strings.ToLower(r.Side))
	v.Set("size", strconv.FormatFloat(r.Size, 'f', -1, 64))

	if r.LimitPrice != nil {
		v.Set("limitPrice", strconv.FormatFloat(*r.LimitPrice, 'f', -1, 64))
	}
	if r.StopPrice != nil {
		v.Set("stopPrice", strconv.FormatFloat(*r.StopPrice, 'f', -1, 64))
	}
	if r.CliOrdID != nil && strings.TrimSpace(*r.CliOrdID) != "" {
		v.Set("cliOrdId", *r.CliOrdID)
	}
	if r.TriggerSignal != nil && strings.TrimSpace(*r.TriggerSignal) != "" {
		v.Set("triggerSignal", strings.ToLower(*r.TriggerSignal))
	}
	if r.ReduceOnly != nil {
		v.Set("reduceOnly", strconv.FormatBool(*r.ReduceOnly))
	}
	if r.ProcessBefore != nil && strings.TrimSpace(*r.ProcessBefore) != "" {
		v.Set("processBefore", *r.ProcessBefore)
	}
	if r.TrailingStopMaxDeviation != nil {
		v.Set("trailingStopMaxDeviation", strconv.FormatFloat(*r.TrailingStopMaxDeviation, 'f', -1, 64))
	}
	if r.TrailingStopDeviationUnit != nil && strings.TrimSpace(*r.TrailingStopDeviationUnit) != "" {
		v.Set("trailingStopDeviationUnit", *r.TrailingStopDeviationUnit)
	}
	if r.LimitPriceOffsetValue != nil {
		v.Set("limitPriceOffsetValue", strconv.FormatFloat(*r.LimitPriceOffsetValue, 'f', -1, 64))
	}
	if r.LimitPriceOffsetUnit != nil && strings.TrimSpace(*r.LimitPriceOffsetUnit) != "" {
		v.Set("limitPriceOffsetUnit", *r.LimitPriceOffsetUnit)
	}
	if r.Broker != nil && strings.TrimSpace(*r.Broker) != "" {
		v.Set("broker", *r.Broker)
	}

	return v, nil
}

type SendOrderResponse struct {
	Result     string `json:"result"`
	ServerTime string `json:"serverTime"`

	SendStatus struct {
		ReceivedTime string `json:"receivedTime"`
		Status       string `json:"status"`
		OrderID      string `json:"order_id"`
	} `json:"sendStatus"`

	// Some responses include orderEvents. Keep it raw unless you need it typed.
	OrderEvents json.RawMessage `json:"orderEvents,omitempty"`
}

func (c *KrakenFuturesClient) SendOrder(req SendOrderRequest) (*SendOrderResponse, error) {
	params, err := req.toValues()
	if err != nil {
		return nil, err
	}

	var out SendOrderResponse
	if err := c.doPrivateRequest("POST", "/sendorder", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Convenience wrapper closer to your Phemex signature.
// ordType examples: "mkt", "ioc", "lmt", "stp", "take_profit".
func (c *KrakenFuturesClient) PlaceOrder(symbol, side string, size float64, ordType string, reduceOnly bool, limitPrice *float64) (*SendOrderResponse, error) {
	clID := fmt.Sprintf("go-%d", time.Now().UnixNano())
	return c.SendOrder(SendOrderRequest{
		OrderType:  ordType,
		Symbol:     symbol,
		Side:       side,
		Size:       size,
		LimitPrice: limitPrice,
		ReduceOnly: &reduceOnly,
		CliOrdID:   &clID,
	})
}

type CancelAllOrdersResponse struct {
	Result     string `json:"result"`
	ServerTime string `json:"serverTime"`

	CancelStatus struct {
		CancelOnly string `json:"cancelOnly"`
		Status     string `json:"status"`
		// cancelledOrders may appear depending on status. Keep raw for flexibility.
		CancelledOrders json.RawMessage `json:"cancelledOrders,omitempty"`
	} `json:"cancelStatus"`

	OrderEvents json.RawMessage `json:"orderEvents,omitempty"`
}

func (c *KrakenFuturesClient) CancelAllOrders(symbol string) (*CancelAllOrdersResponse, error) {
	params := url.Values{}
	if strings.TrimSpace(symbol) != "" {
		params.Set("symbol", symbol)
	}

	var out CancelAllOrdersResponse
	if err := c.doPrivateRequest("POST", "/cancelallorders", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// -----------------------------
// PRIVATE QUERIES
// -----------------------------
type OpenPositionsResponse struct {
	Result     string `json:"result"`
	ServerTime string `json:"serverTime"`

	OpenPositions []OpenPosition `json:"openPositions"`
}

type OpenPosition struct {
	FillTime          string   `json:"fillTime,omitempty"`
	Price             *float64 `json:"price,omitempty"`
	Side              string   `json:"side,omitempty"` // long or short
	Size              float64  `json:"size,omitempty"`
	Symbol            string   `json:"symbol,omitempty"`
	UnrealizedFunding *float64 `json:"unrealizedFunding,omitempty"`
	MaxFixedLeverage  *float64 `json:"maxFixedLeverage,omitempty"`
	PnLCurrency       *string  `json:"pnlCurrency,omitempty"`
}

// GET /openpositions :contentReference[oaicite:5]{index=5}
func (c *KrakenFuturesClient) GetOpenPositions() (*OpenPositionsResponse, error) {
	var out OpenPositionsResponse
	if err := c.doPrivateRequest("GET", "/openpositions", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GET /openorders
func (c *KrakenFuturesClient) GetOpenOrdersRaw() (json.RawMessage, error) {
	var raw json.RawMessage
	// Decode into generic map then re-marshal out if you want. Here we keep it simple.
	var out map[string]any
	if err := c.doPrivateRequest("GET", "/openorders", nil, &out); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(out)
	raw = b
	return raw, nil
}

// GET /fills
func (c *KrakenFuturesClient) GetFillsRaw() (json.RawMessage, error) {
	var out map[string]any
	if err := c.doPrivateRequest("GET", "/fills", nil, &out); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(out)
	return b, nil
}

// CloseAllPositions closes all open positions for a symbol by placing reduceOnly market orders
// on the opposite side. This matches the intent of your Phemex CloseAllPositions.
func (c *KrakenFuturesClient) CloseAllPositions(symbol string) error {
	logger.WithFields(map[string]any{"symbol": symbol}).Info("Closing ALL positions for symbol")

	pos, err := c.GetOpenPositions()
	if err != nil {
		return fmt.Errorf("GetOpenPositions failed: %w", err)
	}

	for _, p := range pos.OpenPositions {
		if strings.TrimSpace(symbol) != "" && p.Symbol != symbol {
			continue
		}
		if p.Size == 0 {
			continue
		}

		var closeSide string
		switch strings.ToLower(p.Side) {
		case "long":
			closeSide = "sell"
		case "short":
			closeSide = "buy"
		default:
			logger.WithFields(map[string]any{"symbol": p.Symbol, "side": p.Side}).Error("Unknown position side, skipping")
			continue
		}

		logger.WithFields(map[string]any{
			"symbol":     p.Symbol,
			"side":       p.Side,
			"size":       p.Size,
			"closeSide":  closeSide,
			"reduceOnly": true,
		}).Info("Closing position")

		// Use "mkt" as the market-style order type in Kraken Futures.
		_, err := c.PlaceOrder(p.Symbol, closeSide, p.Size, "mkt", true, nil)
		if err != nil {
			return fmt.Errorf("failed to close position %s (%s) size=%f: %w", p.Symbol, p.Side, p.Size, err)
		}
	}

	logger.WithFields(map[string]any{"symbol": symbol}).Info("All positions successfully closed")
	return nil
}

// -----------------------------
// PUBLIC MARKET DATA (OPTIONAL)
// -----------------------------
type TickerBySymbolResponse struct {
	Result     string `json:"result"`
	ServerTime string `json:"serverTime"`
	Ticker     any    `json:"ticker"` // keep flexible
}

// GET /tickers/:symbol :contentReference[oaicite:6]{index=6}
func (c *KrakenFuturesClient) GetTickerBySymbol(symbol string) (*TickerBySymbolResponse, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, errors.New("symbol is required")
	}
	var out TickerBySymbolResponse
	ep := "/tickers/" + url.PathEscape(symbol)
	if err := c.doPublicRequest("GET", ep, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type OrderbookResponse struct {
	Result     string `json:"result"`
	ServerTime string `json:"serverTime"`
	OrderBook  any    `json:"orderBook"` // keep flexible
}

// GET /orderbook?symbol=... :contentReference[oaicite:7]{index=7}
func (c *KrakenFuturesClient) GetOrderbook(symbol string) (*OrderbookResponse, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, errors.New("symbol is required")
	}
	params := url.Values{}
	params.Set("symbol", symbol)

	var out OrderbookResponse
	if err := c.doPublicRequest("GET", "/orderbook", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
