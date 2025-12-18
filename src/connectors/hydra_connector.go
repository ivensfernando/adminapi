package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const userAgentDefault = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"

type GooeyClient struct {
	BaseURL              *url.URL
	HTTP                 *http.Client
	SessionCookie        *http.Cookie // e.g. JSESSIONID
	DxtfidCookie         *http.Cookie // optional. DXTFID if present
	CSRFTok              string
	AtmosphereTrackingID string
	UserAgent            string
	positionsMu          sync.RWMutex
	positions            map[string]Position // key: accountId:positionCode
	APIKey               string
	APISecret            string
}

func NewGooeyClient(apiKey, apiSecret string) (*GooeyClient, error) {
	u, err := url.Parse("https://trade.gooeytrade.com")
	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &GooeyClient{
		BaseURL: u,
		HTTP: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
		APIKey:    apiKey,
		APISecret: apiSecret,
	}, nil
}

// Login posts credentials. stores any cookies that come back.
func (c *GooeyClient) Login(ctx context.Context) error {
	loginURL := c.BaseURL.ResolveReference(&url.URL{Path: "/api/auth/login"}).String()

	payload := map[string]string{
		"vendor":   "fptech",
		"username": c.APIKey,
		"password": c.APISecret,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", c.BaseURL.String())
	req.Header.Set("Referer", c.BaseURL.String()+"/")
	req.Header.Set("User-Agent", userAgentDefault)
	req.Header.Set("Accept", "*/*")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("login non-2xx status: %d", resp.StatusCode)
	}

	// Pull cookies out of the jar and store explicit references
	c.syncCookiesFromJar()
	if c.SessionCookie == nil {
		return errors.New("JSESSIONID not set after login")
	}
	return nil
}

func (c *GooeyClient) syncCookiesFromJar() {
	if c.HTTP == nil || c.HTTP.Jar == nil || c.BaseURL == nil {
		return
	}
	cookies := c.HTTP.Jar.Cookies(c.BaseURL)
	for _, ck := range cookies {
		switch ck.Name {
		case "JSESSIONID":
			c.SessionCookie = ck
		case "DXTFID":
			c.DxtfidCookie = ck
		}
	}
}

var reCSRF = regexp.MustCompile(`(?i)<meta[^>]+id=["']csrf-token["'][^>]*content=["']([^"']+)["']`)

// FetchCSRF loads the root HTML and extracts the meta csrf token. stores it.
func (c *GooeyClient) FetchCSRF(ctx context.Context) error {
	rootURL := c.BaseURL.ResolveReference(&url.URL{Path: "/"}).String()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rootURL, nil)
	req.Header.Set("User-Agent", userAgentDefault)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// Attach stored cookies explicitly. net/http will also attach via Jar, but we do it explicitly
	// to match your requirement to store and reuse cookies.
	if c.SessionCookie != nil {
		req.AddCookie(c.SessionCookie)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("fetch root failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("root non-2xx status: %d", resp.StatusCode)
	}

	// Jar will absorb any Set-Cookie. now sync into struct fields
	c.syncCookiesFromJar()

	b, _ := io.ReadAll(resp.Body)
	m := reCSRF.FindSubmatch(b)
	if len(m) < 2 {
		return errors.New("csrf meta tag not found")
	}
	c.CSRFTok = string(m[1])
	return nil
}

type OrderSide string

const (
	SideBuy  OrderSide = "BUY"
	SideSell OrderSide = "SELL"
)

type PositionEffect string

const (
	PositionOpen  PositionEffect = "OPENING"
	PositionClose PositionEffect = "CLOSING"
)

type OrderOption func(*Order)

func WithStopLoss(price, offset, qty float64) OrderOption {
	return func(o *Order) {
		o.StopLoss.FixedPrice = price
		o.StopLoss.FixedOffset = offset
		o.StopLoss.OrderType = "STOP"
		o.StopLoss.PriceFixed = true
		o.StopLoss.QuantityForProtection = qty
		o.StopLoss.Removed = false
	}
}

//
//// WithPercentStopLoss Convenience. compute a percent stop and delegate to WithStopLoss.
//func WithPercentStopLoss(entry float64, percent float64, side string) OrderOption {
//	return func(o *Order) {
//		pct := percent / 100.0
//
//		var slPrice float64
//		switch side {
//		case "buy":
//			// long. stop below entry
//			slPrice = entry * (1 - pct)
//		case "sell ":
//			// short. stop above entry
//			slPrice = entry * (1 + pct)
//		default:
//			// invalid side. do nothing
//			return
//		}
//
//		offset := math.Abs(entry - slPrice)
//		WithStopLoss(slPrice, offset)(o)
//	}
//}

func CalcStopLoss(entry float64, percent float64, side string) float64 {
	pct := percent / 100.0

	switch side {
	case "buy":
		return entry * (1 - pct)
	case "sell":
		return entry * (1 + pct)
	default:
		panic("invalid side")
	}
}

// generateRequestID creates DXTrade-style request IDs:
// gwt-uid-<4-digit-int>-<uuid>
func generateRequestID() string {
	// Local RNG instance. avoids global rand.Seed.
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	// 1000–9999 range to match DXTrade patterns like "1115"
	r := rng.Intn(9000) + 1000

	return fmt.Sprintf("gwt-uid-%d-%s", r, uuid.NewString())
}

func WithRequestID(id string) OrderOption {
	return func(o *Order) {
		if id == "" {
			id = generateRequestID()
		}
		o.RequestID = id
	}
}

func WithLimitPrice(price float64) OrderOption {
	return func(o *Order) { o.LimitPrice = price }
}

func (c *GooeyClient) PlaceMarketOrder(
	ctx context.Context,
	instrumentID int,
	symbol string,
	quantity float64,
	side OrderSide,
	effect PositionEffect,
	opts ...OrderOption,
) ([]byte, int, error) {

	var ord Order
	ord.DirectExchange = false
	ord.OrderType = "MARKET"
	ord.OrderSide = string(side)
	ord.Quantity = quantity
	ord.TimeInForce = "GTC"

	ord.Legs = []struct {
		InstrumentID   int    `json:"instrumentId"`
		PositionEffect string `json:"positionEffect"`
		RatioQuantity  int    `json:"ratioQuantity"`
		Symbol         string `json:"symbol"`
	}{
		{
			InstrumentID:   instrumentID,
			PositionEffect: string(effect),
			RatioQuantity:  1,
			Symbol:         symbol,
		},
	}

	// Default request ID unless overridden
	if ord.RequestID == "" {
		ord.RequestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}

	// Apply functional options
	for _, opt := range opts {
		opt(&ord)
	}

	// Send request
	resp, status, err := c.PostJSON(ctx, "/api/orders/single", ord)
	if err != nil {
		return resp, status, fmt.Errorf("PlaceMarketOrder failed: %w", err)
	}
	return resp, status, nil
}

func (c *GooeyClient) PostJSON(ctx context.Context, path string, payload any) ([]byte, int, error) {
	u := c.BaseURL.ResolveReference(&url.URL{Path: path}).String()
	buf, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(buf))
	req.Header.Set("User-Agent", userAgentDefault)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Origin", c.BaseURL.String())
	req.Header.Set("Referer", c.BaseURL.String()+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// Required headers and cookies
	if c.CSRFTok != "" {
		req.Header.Set("X-CSRF-Token", c.CSRFTok)
	}
	if c.DxtfidCookie != nil {
		req.AddCookie(c.DxtfidCookie)
	}
	if c.SessionCookie != nil {
		req.AddCookie(c.SessionCookie)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return out, resp.StatusCode, nil
}

type Order struct {
	DirectExchange bool `json:"directExchange"`
	Legs           []struct {
		InstrumentID   int    `json:"instrumentId"`
		PositionEffect string `json:"positionEffect"`
		RatioQuantity  int    `json:"ratioQuantity"`
		Symbol         string `json:"symbol"`
	} `json:"legs"`
	LimitPrice  float64 `json:"limitPrice"`
	OrderSide   string  `json:"orderSide"`
	OrderType   string  `json:"orderType"`
	Quantity    float64 `json:"quantity"`
	RequestID   string  `json:"requestId"`
	TimeInForce string  `json:"timeInForce"`
	StopLoss    struct {
		FixedOffset           float64 `json:"fixedOffset"`
		FixedPrice            float64 `json:"fixedPrice"`
		OrderType             string  `json:"orderType"`
		PriceFixed            bool    `json:"priceFixed"`
		QuantityForProtection float64 `json:"quantityForProtection"`
		Removed               bool    `json:"removed"`
	} `json:"stopLoss"`
}

// ClosePosition posts to /api/positions/close using the stored cookies and X-CSRF-Token.
func (c *GooeyClient) ClosePosition(ctx context.Context, legs []map[string]any,
	limitPrice float64, orderType string, quantity float64, tif string) ([]byte, int, error) {
	link := c.BaseURL.ResolveReference(&url.URL{Path: "/api/positions/close"}).String()

	payload := map[string]any{
		"legs":        legs,
		"limitPrice":  limitPrice,
		"orderType":   orderType, // e.g. "MARKET"
		"quantity":    quantity,  // negative for sell/close
		"timeInForce": tif,       // e.g. "GTC"
	}

	buf, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, link, bytes.NewReader(buf))
	req.Header.Set("User-Agent", userAgentDefault)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Origin", c.BaseURL.String())
	req.Header.Set("Referer", c.BaseURL.String()+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if c.CSRFTok != "" {
		req.Header.Set("X-CSRF-Token", c.CSRFTok)
	}
	// optional Atmosphere tracking if your session requires it
	//req.Header.Set("X-Atmosphere-tracking-id", c.AtmosphereTrackingID)

	// cookies from session
	if c.DxtfidCookie != nil {
		req.AddCookie(c.DxtfidCookie)
	}
	if c.SessionCookie != nil {
		req.AddCookie(c.SessionCookie)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("close position failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// Trade represents a single trade history entry from /api/trades/history
type Trade struct {
	Time              int64              `json:"time"`
	AccountID         int64              `json:"accountId"`
	Symbol            string             `json:"symbol"`
	OrderChainID      int64              `json:"orderChainId"`
	TradeID           int64              `json:"tradeId"`
	TradeCode         string             `json:"tradeCode"`
	TradeSide         string             `json:"tradeSide"`
	PositionEffect    string             `json:"positionEffect"`
	Quantity          float64            `json:"quantity"`
	FillPrice         float64            `json:"fillPrice"`
	FullCost          string             `json:"fullCost"`
	FullCostPrecision int                `json:"fullCostPrecision"`
	FullCostCurrency  string             `json:"fullCostCurrency"`
	Commission        map[string]float64 `json:"commission"`
	//SettledPL          string             `json:"settledPL"`
	//NetPL              string `json:"netPl"`
	//SettledPLPips      string `json:"settledPlPips"`
	SettledPLPrecision int    `json:"settledPLPrecision"`
	SettledPLCurrency  string `json:"settledPLCurrency"`
}

// HistoryTrades queries POST /api/trades/history?from=...&to=...
// fromMs and toMs are epoch millis.
func (c *GooeyClient) HistoryTrades(ctx context.Context, fromMs, toMs int64) ([]Trade, int, error) {
	// build URL with query params
	u := c.BaseURL.ResolveReference(&url.URL{
		Path:     "/api/trades/history",
		RawQuery: fmt.Sprintf("from=%d&to=%d", fromMs, toMs),
	}).String()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, http.NoBody)
	req.Header.Set("User-Agent", userAgentDefault)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Origin", c.BaseURL.String())
	req.Header.Set("Referer", c.BaseURL.String()+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if c.CSRFTok != "" {
		req.Header.Set("X-CSRF-Token", c.CSRFTok)
	}
	// optional. use a stable or generated tracking id if your session needs it
	//req.Header.Set("X-Atmosphere-tracking-id", c.AtmosphereTrackingID)

	// attach stored cookies
	if c.DxtfidCookie != nil {
		req.AddCookie(c.DxtfidCookie)
	}
	if c.SessionCookie != nil {
		req.AddCookie(c.SessionCookie)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("history trades request failed: %w", err)
	}
	defer resp.Body.Close()

	status := resp.StatusCode
	data, _ := io.ReadAll(resp.Body)

	if status/100 != 2 {
		return nil, status, fmt.Errorf("history trades non 2xx. status=%d body=%s", status, string(data))
	}

	var trades []Trade
	if err := json.Unmarshal(data, &trades); err != nil {
		return nil, status, fmt.Errorf("decode trades failed: %w", err)
	}
	return trades, status, nil
}

func ToMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

type TradeJournalEntry struct {
	Date            int64    `json:"date"`
	PositionComment string   `json:"positionComment"`
	PositionSide    string   `json:"positionSide"`
	TradeHistoryTO  Trade    `json:"tradeHistoryTO"`
	PositionCode    string   `json:"positionCode"` // e.g. "#9910/295152261"
	AccountID       int64    `json:"accountId"`
	TradeID         int64    `json:"tradeId"`
	TradeComment    string   `json:"tradeComment"`
	TradeTags       []string `json:"tradeTags"`
}

// TradeJournal calls POST /api/tradejournal?from=...&to=...
func (c *GooeyClient) TradeJournal(ctx context.Context, fromMs, toMs int64) ([]TradeJournalEntry, int, error) {
	u := c.BaseURL.ResolveReference(&url.URL{
		Path:     "/api/tradejournal",
		RawQuery: fmt.Sprintf("from=%d&to=%d", fromMs, toMs),
	}).String()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	//req.Header.Set("User-Agent", userAgentDefault)
	//req.Header.Set("Accept", "*/*")
	//req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	//req.Header.Set("Origin", c.BaseURL.String())
	//req.Header.Set("Referer", c.BaseURL.String()+"/")

	//// sec-ch-ua* are not strictly necessary for the backend, but harmless to mirror
	//req.Header.Set("sec-ch-ua", `"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"`)
	//req.Header.Set("sec-ch-ua-mobile", "?0")
	//req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	//
	//req.Header.Set("Sec-Fetch-Dest", "empty")
	//req.Header.Set("Sec-Fetch-Mode", "cors")
	//req.Header.Set("Sec-Fetch-Site", "same-origin")

	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	//if c.CSRFTok != "" {
	//	req.Header.Set("X-CSRF-Token", c.CSRFTok)
	//}
	//req.Header.Set("X-Atmosphere-tracking-id", c.AtmosphereTrackingID)

	if c.DxtfidCookie != nil {
		req.AddCookie(c.DxtfidCookie)
	}
	if c.SessionCookie != nil {
		req.AddCookie(c.SessionCookie)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("tradejournal request failed: %w", err)
	}
	defer resp.Body.Close()

	status := resp.StatusCode
	data, _ := io.ReadAll(resp.Body)
	if status/100 != 2 {
		return nil, status, fmt.Errorf("tradejournal non-2xx. status=%d body=%s", status, string(data))
	}

	var entries []TradeJournalEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, status, fmt.Errorf("decode tradejournal failed: %w", err)
	}
	return entries, status, nil
}

type posKey struct {
	AccountID    int64
	PositionCode string // full "#9910/295152261"
}

type OpenPositionFromJournal struct {
	AccountID    int64
	Symbol       string
	InstrumentID int
	PositionCode string // numeric, e.g. "295152261"
	NetQty       float64
	LastPrice    float64 // last fill price seen in journal
}

func computeOpenFromJournal(entries []TradeJournalEntry) ([]OpenPositionFromJournal, error) {
	net := make(map[posKey]float64)
	meta := make(map[posKey]OpenPositionFromJournal)

	for _, e := range entries {
		th := e.TradeHistoryTO
		k := posKey{AccountID: e.AccountID, PositionCode: e.PositionCode}

		net[k] += th.Quantity // BUY positive, SELL negative

		// keep last meta. we parse instrumentId & numeric positionCode here
		raw := strings.TrimPrefix(e.PositionCode, "#")
		parts := strings.SplitN(raw, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected positionCode format: %s", e.PositionCode)
		}
		instID, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("bad instrumentId in positionCode %s: %w", e.PositionCode, err)
		}
		meta[k] = OpenPositionFromJournal{
			AccountID:    e.AccountID,
			Symbol:       th.Symbol,
			InstrumentID: instID,
			PositionCode: parts[1], // numeric code only
			LastPrice:    th.FillPrice,
		}
	}

	var result []OpenPositionFromJournal
	const eps = 1e-12
	for k, qty := range net {
		if math.Abs(qty) < eps {
			continue // fully closed
		}
		m := meta[k]
		m.NetQty = qty
		result = append(result, m)
	}
	return result, nil
}

func (c *GooeyClient) CloseAllOpenFromTradeJournal(ctx context.Context, from, to time.Time) error {
	fmt.Println("From:", from.Format(time.RFC3339))
	fmt.Println("To:  ", to.Format(time.RFC3339))

	fromMs := ToMillis(from)
	toMs := ToMillis(to)

	journal, status, err := c.TradeJournal(ctx, fromMs, toMs)
	if err != nil {
		return fmt.Errorf("tradejournal error: %w", err)
	}
	fmt.Println("tradejournal HTTP status:", status)

	for _, e := range journal {
		t := time.UnixMilli(e.TradeHistoryTO.Time).UTC()
		fmt.Printf("[%s] %s %s posCode=%s qty=%f price=%f effect=%s\n",
			t.Format(time.RFC3339),
			e.TradeHistoryTO.TradeSide,
			e.TradeHistoryTO.Symbol,
			e.PositionCode,
			e.TradeHistoryTO.Quantity,
			e.TradeHistoryTO.FillPrice,
			e.TradeHistoryTO.PositionEffect,
		)
	}

	openPositions, err := computeOpenFromJournal(journal)
	if err != nil {
		return err
	}
	if len(openPositions) == 0 {
		fmt.Println("No open positions found in tradejournal range.")
		return nil
	}

	for _, p := range openPositions {
		// send the inverse to flatten
		closeQty := -p.NetQty

		legs := []map[string]any{
			{
				"instrumentId":   p.InstrumentID,
				"positionCode":   p.PositionCode, // numeric only
				"positionEffect": "CLOSING",
				"ratioQuantity":  1,
				"symbol":         p.Symbol,
			},
		}

		fmt.Printf("Closing position. acc=%d symbol=%s instr=%d posCode=%s netQty=%f closeQty=%f\n",
			p.AccountID, p.Symbol, p.InstrumentID, p.PositionCode, p.NetQty, closeQty)

		// MARKET orders in your examples carry a limitPrice, but backend usually ignores it.
		// You can pass 0 or last fill price. I’ll use last fill price for clarity.
		limitPrice := p.LastPrice

		resp, status, err := c.ClosePosition(ctx, legs, limitPrice, "MARKET", closeQty, "GTC")
		if err != nil {
			fmt.Printf("ClosePosition error for %s (%s): %v\n", p.Symbol, p.PositionCode, err)
			continue
		}
		fmt.Printf("ClosePosition status=%d body=%s\n", status, string(resp))
	}

	return nil
}

type Position struct {
	UID         string `json:"uid"`
	AccountID   string `json:"accountId"`
	PositionKey struct {
		InstrumentID int    `json:"instrumentId"`
		PositionCode string `json:"positionCode"`
	} `json:"positionKey"`
	Quantity     float64 `json:"quantity"`
	Cost         float64 `json:"cost"`
	CostBasis    float64 `json:"costBasis"`
	OpenCost     float64 `json:"openCost"`
	MarginRate   float64 `json:"marginRate"`
	Time         int64   `json:"time"`
	ModifiedTime int64   `json:"modifiedTime"`
	UserLogin    string  `json:"userLogin"`
	// takeProfit, stopLoss omitted for brevity
}

type WSMessage struct {
	Type      string          `json:"type"`
	AccountID *string         `json:"accountId"`
	BodyRaw   json.RawMessage `json:"body"`
}

func (c *GooeyClient) handlePositions(body json.RawMessage) {
	var positions []Position
	if err := json.Unmarshal(body, &positions); err != nil {
		log.Println("POSITIONS decode error:", err)
		return
	}

	c.positionsMu.Lock()
	defer c.positionsMu.Unlock()

	for _, p := range positions {
		key := fmt.Sprintf("%s:%s", p.AccountID, p.PositionKey.PositionCode)

		// quantity == 0 means closed. remove from map
		if math.Abs(p.Quantity) < 1e-12 {
			delete(c.positions, key)
			continue
		}
		c.positions[key] = p
	}
}

//
//func (c *GooeyClient) RunWSConsumer(ctx context.Context, conn *websocket.Conn) {
//	defer conn.Close()
//
//	for {
//		select {
//		case <-ctx.Done():
//			log.Println("WS consumer stopping:", ctx.Err())
//			return
//		default:
//		}
//
//		_, msg, err := conn.ReadMessage()
//		if err != nil {
//			log.Println("WS read error:", err)
//			return
//		}
//
//		pipeIdx := bytes.IndexByte(msg, '|')
//		if pipeIdx > 0 {
//			msg = msg[pipeIdx+1:]
//		}
//		msg = bytes.TrimSpace(msg)
//		if len(msg) == 0 || msg[0] != '{' {
//			continue
//		}
//
//		var base WSMessage
//		if err := json.Unmarshal(msg, &base); err != nil {
//			log.Println("WS json unmarshal error:", err, "raw:", string(msg))
//			continue
//		}
//
//		switch base.Type {
//		case "POSITIONS":
//			c.handlePositions(base.BodyRaw)
//		//case "INSTRUMENT_METRICS":
//		//	// optionally handle. not needed for closing
//		//case "POSITION_METRICS":
//		//	// optional
//		//case "QUOTE":
//		//	handleQuote(base.BodyRaw)
//		//case "SUMMARY":
//		//	handleSummary(base.BodyRaw)
//		//case "chartFeedSubtopic":
//		//	handleChartFeed(base.BodyRaw)
//		//case "CONVERSION_RATE":
//		//	handleConversion(base.BodyRaw)
//		default:
//			// debug only if needed
//			// log.Println("WS unknown type:", base.Type)
//		}
//	}
//}

func (c *GooeyClient) InitAtmosphereTrackingID(ctx context.Context) error {
	// If we already have a tracking id, reuse it
	if c.AtmosphereTrackingID != "" {
		return nil
	}

	wsURL := url.URL{
		Scheme: "wss",
		Host:   c.BaseURL.Host, // trade.gooeytrade.com
		Path:   "/client/connector",
		RawQuery: url.Values{
			"X-Atmosphere-tracking-id":      []string{"0"},
			"X-Atmosphere-Framework":        []string{"2.3.2-javascript"},
			"X-Atmosphere-Transport":        []string{"websocket"},
			"X-Atmosphere-TrackMessageSize": []string{"true"},
			"Content-Type":                  []string{"text/x-gwt-rpc; charset=UTF-8"},
			"X-atmo-protocol":               []string{"true"},
			"sessionState":                  []string{"dx-new"},
			"guest-mode":                    []string{"false"},
		}.Encode(),
	}

	header := http.Header{}
	header.Set("Origin", c.BaseURL.String())
	header.Set("User-Agent", c.UserAgent)
	header.Set("Cache-Control", "no-cache")
	header.Set("Pragma", "no-cache")
	header.Set("Accept-Language", "en-GB,en-US;q=0.9,en;q=0.8")

	// Attach cookies
	var cookieVals []string
	if c.SessionCookie != nil {
		cookieVals = append(cookieVals, c.SessionCookie.String())
	}
	if c.DxtfidCookie != nil {
		cookieVals = append(cookieVals, c.DxtfidCookie.String())
	}
	if len(cookieVals) > 0 {
		header.Set("Cookie", strings.Join(cookieVals, "; "))
	}

	dialer := websocket.Dialer{
		HandshakeTimeout:  15 * time.Second,
		EnableCompression: true,
		Proxy:             http.ProxyFromEnvironment,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL.String(), header)
	if err != nil {
		return fmt.Errorf("ws dial failed: %w", err)
	}
	defer conn.Close()

	// First frame looks like: "41|<uuid>|0|X|"
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("ws read failed: %w", err)
	}

	raw := string(msg)
	parts := strings.Split(raw, "|")
	if len(parts) < 2 {
		return fmt.Errorf("unexpected ws handshake payload: %q", raw)
	}
	id := parts[1]
	if id == "" {
		return fmt.Errorf("empty tracking id in ws payload: %q", raw)
	}

	c.AtmosphereTrackingID = id
	return nil
}
