package connectors

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
)

// ---------------------------------------------------------------------
// CONFIG BÁSICO
// ---------------------------------------------------------------------

const (
	kucoinSpotBaseURL    = "https://api.kucoin.com"
	kucoinFuturesBaseURL = "https://api-futures.kucoin.com"
	httpTimeout          = 10 * time.Second
)

// ---------------------------------------------------------------------
// TIPOS DE APOIO
// ---------------------------------------------------------------------

// Resposta genérica da KuCoin
type kucoinAPIResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg,omitempty"`
	Data json.RawMessage `json:"data"`
}

// KucoinFuturesContract representa os campos principais de /api/v1/contracts/{symbol}
type KucoinFuturesContract struct {
	Symbol          string  `json:"symbol"`
	RootSymbol      string  `json:"rootSymbol"`
	Type            string  `json:"type"` // "FFWCSX" etc.
	BaseCurrency    string  `json:"baseCurrency"`
	QuoteCurrency   string  `json:"quoteCurrency"`
	SettleCurrency  string  `json:"settleCurrency"`
	Multiplier      float64 `json:"multiplier"`     // tamanho do contrato
	MultiplierCoin  string  `json:"multiplierCoin"` // ex: "XBT"
	MaxLeverage     float64 `json:"maxLeverage"`
	LotSize         float64 `json:"lotSize"`
	TickSize        float64 `json:"tickSize"`
	PricePrecision  int     `json:"pricePrecision"`
	VolumePrecision int     `json:"volumePrecision"`
	FundingFeeRate  float64 `json:"fundingFeeRate"`
	PremiumsSymbol  string  `json:"premiumsSymbol"`
	MarginCurrency  string  `json:"marginCurrency"`
	IsInverse       bool    `json:"isInverse"`
	MaintMarginRate float64 `json:"maintainMarginRate"`
	MarkMethod      string  `json:"markMethod"`
	RiskStep        float64 `json:"riskStep"`
	MinRiskLimit    float64 `json:"minRiskLimit"`
	MaxRiskLimit    float64 `json:"maxRiskLimit"`
	RiskLimitStep   float64 `json:"riskLimitStep"`
	// podes adicionar mais campos conforme fores precisando
}

// Spot account (GET /api/v1/accounts)
type kucoinSpotAccount struct {
	ID        string `json:"id"`
	Currency  string `json:"currency"`
	Type      string `json:"type"`    // main / trade / ...
	Balance   string `json:"balance"` // string numérica
	Available string `json:"available"`
	Holds     string `json:"holds"`
}

// Futures account overview (GET /api/v1/account-overview)
type kucoinFuturesAccountOverview struct {
	AccountEquity     float64 `json:"accountEquity"`
	UnrealisedPNL     float64 `json:"unrealisedPNL"`
	MarginBalance     float64 `json:"marginBalance"`
	PositionMargin    float64 `json:"positionMargin"`
	OrderMargin       float64 `json:"orderMargin"`
	FrozenFunds       float64 `json:"frozenFunds"`
	AvailableBalance  float64 `json:"availableBalance"`
	Currency          string  `json:"currency"`
	RiskRatio         float64 `json:"riskRatio"`
	MaxWithdrawAmount float64 `json:"maxWithdrawAmount"`
}

func toFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case uint64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

// ---------------------------------------------------------------------
// ASSINATURAS (MESMO ESQUEMA DO TEU main.go)
// ---------------------------------------------------------------------

// KC-API-PASSPHRASE = base64( HMAC_SHA256(apiSecret, apiPassphrase) )
func kucoinSignPassphrase(secret, passphrase string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(passphrase))
	hash := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash)
}

// KC-API-SIGN = base64( HMAC_SHA256(apiSecret, timestamp + method + requestPath + body) )
// requestPath = path + queryString (ex: "/api/v1/accounts?type=trade")
func kucoinSignRequest(secret, timestamp, method, requestPath, body string) string {
	prehash := timestamp + method + requestPath + body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(prehash))
	hash := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash)
}

// ---------------------------------------------------------------------
// CLIENTE BAIXO NÍVEL (SPOT OU FUTURES)
// ---------------------------------------------------------------------

type kucoinRESTClient struct {
	apiKey        string
	apiSecret     string
	apiPassphrase string
	keyVersion    string
	baseURL       string
	httpClient    *http.Client
}

func newKucoinRESTClient(
	apiKey, apiSecret, apiPassphrase, keyVersion, baseURL string,
) *kucoinRESTClient {
	return &kucoinRESTClient{
		apiKey:        apiKey,
		apiSecret:     apiSecret,
		apiPassphrase: apiPassphrase,
		keyVersion:    keyVersion,
		baseURL:       baseURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// doRequest performs a signed HTTP call to KuCoin and returns a parsed kucoinAPIResponse.
func (c *kucoinRESTClient) doRequest(
	method, endpoint, query, body string,
) (*kucoinAPIResponse, error) {

	// Build request path used for signing (path + query)
	requestPath := endpoint
	if query != "" {
		requestPath = endpoint + "?" + query
	}

	// Full URL for logging
	fullURL := c.baseURL + requestPath

	// Timestamp in ms
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond))

	// Calculate request signature
	signature := kucoinSignRequest(c.apiSecret, timestamp, method, requestPath, body)

	// Encrypted passphrase
	encryptedPassphrase := kucoinSignPassphrase(c.apiSecret, c.apiPassphrase)

	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewBuffer([]byte(body))
	} else {
		bodyReader = nil
	}

	// Log outgoing request
	logger.WithFields(logger.Fields{
		"method": method,
		"url":    fullURL,
		"body":   body,
	}).Debug("KuCoin HTTP request")

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		logger.WithError(err).Error("Failed to create KuCoin HTTP request")
		return nil, fmt.Errorf("new request: %w", err)
	}

	// Required KuCoin headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("KC-API-KEY", c.apiKey)
	req.Header.Set("KC-API-SIGN", signature)
	req.Header.Set("KC-API-TIMESTAMP", timestamp)
	req.Header.Set("KC-API-PASSPHRASE", encryptedPassphrase)
	if c.keyVersion != "" {
		req.Header.Set("KC-API-KEY-VERSION", c.keyVersion) // e.g. "3"
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.WithError(err).Error("KuCoin HTTP request failed")
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Error("Failed to read KuCoin HTTP response body")
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Log raw HTTP status & body
	logger.WithFields(logger.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("KuCoin HTTP response")

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		logger.WithFields(logger.Fields{
			"status": resp.StatusCode,
			"body":   string(respBody),
		}).Error("KuCoin HTTP non-2xx status")
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp kucoinAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		logger.WithError(err).Error("Failed to unmarshal KuCoin API response")
		return nil, fmt.Errorf("unmarshal kucoin response: %w", err)
	}

	if apiResp.Code != "200000" {
		logger.WithFields(logger.Fields{
			"code": apiResp.Code,
			"msg":  apiResp.Msg,
		}).Error("KuCoin API returned error code")
		return nil, fmt.Errorf("kucoin error code=%s msg=%s", apiResp.Code, apiResp.Msg)
	}

	return &apiResp, nil
}

// ---------------------------------------------------------------------
// CONNECTOR DE ALTO NÍVEL (SPOT + FUTURES)
// ---------------------------------------------------------------------

type KucoinConnector struct {
	spotClient    *kucoinRESTClient
	futuresClient *kucoinRESTClient
}

// NewKucoinConnector cria um connector usando REST cru (sem ccxt).
func NewKucoinConnector(
	apiKey, apiSecret, apiPassphrase, keyVersion string,
) *KucoinConnector {
	return &KucoinConnector{
		spotClient:    newKucoinRESTClient(apiKey, apiSecret, apiPassphrase, keyVersion, kucoinSpotBaseURL),
		futuresClient: newKucoinRESTClient(apiKey, apiSecret, apiPassphrase, keyVersion, kucoinFuturesBaseURL),
	}
}

// TestConnection checks if we can reach both spot and futures APIs.
func (k *KucoinConnector) TestConnection() error {
	logger.Info("Testing KuCoin spot and futures connectivity")

	if _, err := k.spotClient.doRequest(http.MethodGet, "/api/v1/accounts", "", ""); err != nil {
		logger.WithError(err).Error("KuCoin spot ping failed")
		return fmt.Errorf("spot ping failed: %w", err)
	}

	if _, err := k.futuresClient.doRequest(http.MethodGet, "/api/v1/account-overview", "currency=USDT", ""); err != nil {
		logger.WithError(err).Error("KuCoin futures ping failed")
		return fmt.Errorf("futures ping failed: %w", err)
	}

	logger.Info("KuCoin spot and futures ping succeeded")
	return nil
}

// GetAccountBalances aggregates spot and futures balances into a simple map.
func (k *KucoinConnector) GetAccountBalances() (map[string]float64, error) {
	logger.Info("Fetching KuCoin spot and futures balances")

	balances := make(map[string]float64)

	// Spot balances: GET /api/v1/accounts
	spotResp, err := k.spotClient.doRequest(http.MethodGet, "/api/v1/accounts", "", "")
	if err != nil {
		logger.WithError(err).Error("Failed to fetch KuCoin spot balances")
		return nil, fmt.Errorf("fetch spot balances: %w", err)
	}

	var spotAccounts []kucoinSpotAccount
	if err := json.Unmarshal(spotResp.Data, &spotAccounts); err != nil {
		logger.WithError(err).Error("Failed to unmarshal KuCoin spot accounts")
		return nil, fmt.Errorf("unmarshal spot accounts: %w", err)
	}

	for _, acc := range spotAccounts {
		if acc.Available == "" {
			continue
		}
		avail, err := strconv.ParseFloat(acc.Available, 64)
		if err != nil || avail == 0 {
			continue
		}
		key := fmt.Sprintf("spot_%s", acc.Currency)
		balances[key] += avail
	}

	// Futures balance: GET /api/v1/account-overview?currency=USDT
	futuresResp, err := k.futuresClient.doRequest(
		http.MethodGet,
		"/api/v1/account-overview",
		"currency=USDT",
		"",
	)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch KuCoin futures balance")
		return nil, fmt.Errorf("fetch futures balance: %w", err)
	}

	var fut kucoinFuturesAccountOverview
	if err := json.Unmarshal(futuresResp.Data, &fut); err != nil {
		logger.WithError(err).Error("Failed to unmarshal KuCoin futures account overview")
		return nil, fmt.Errorf("unmarshal futures account: %w", err)
	}

	balances[fmt.Sprintf("futures_%s", fut.Currency)] = fut.AvailableBalance

	logger.WithField("balances", balances).Info("KuCoin balances fetched successfully")
	return balances, nil
}

// GetAvailableBaseFromUSDT converts the available USDT balance into base units using the latest ticker price.
func (k *KucoinConnector) GetAvailableBaseFromUSDT(
	symbol string,
) (baseSymbol string, baseAvail float64, usdtAvail float64, price float64, err error) {

	if symbol == "" {
		err = fmt.Errorf("symbol is required")
		return
	}

	baseSymbol = symbol
	switch {
	case strings.HasSuffix(symbol, "USDTM"):
		baseSymbol = strings.TrimSuffix(symbol, "USDTM")
	case strings.HasSuffix(symbol, "USDT"):
		baseSymbol = strings.TrimSuffix(symbol, "USDT")
	}

	usdtAvail, err = k.GetFuturesAvailableFromRiskUnit(symbol)
	if err != nil {
		return
	}

	ticker, err := k.GetFuturesTicker(symbol)
	if err != nil {
		return
	}

	price = toFloat(ticker["price"])
	if price <= 0 {
		price = toFloat(ticker["lastTradePrice"])
	}
	if price <= 0 {
		err = fmt.Errorf("invalid price for %s", symbol)
		return
	}

	baseAvail = usdtAvail / price
	return
}

// ---------------------------------------------------------------------
// EXEMPLO DE EXECUÇÃO DE ORDEM FUTUROS (opcional, se quiseres já deixar pronto)
// ---------------------------------------------------------------------

// PlaceFuturesOrder sends a futures order to KuCoin.
func (k *KucoinConnector) PlaceFuturesOrder(
	symbol string,
	side string,
	orderType string,
	size int64,
	reduceOnly bool,
) (*kucoinAPIResponse, error) {

	if symbol == "" || side == "" {
		return nil, fmt.Errorf("symbol and side are required")
	}
	if size <= 0 {
		return nil, fmt.Errorf("order size must be greater than zero")
	}

	clientOid := fmt.Sprintf("go-%d", time.Now().UnixNano())

	body := map[string]interface{}{
		"clientOid":  clientOid,
		"symbol":     symbol,
		"side":       side,
		"type":       orderType,
		"size":       size,
		"reduceOnly": reduceOnly,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal order body: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol": symbol,
		"side":   side,
		"type":   orderType,
		"size":   size,
	}).Info("Placing KuCoin futures order")

	return k.futuresClient.doRequest(
		http.MethodPost,
		"/api/v1/orders",
		"",
		string(b),
	)
}

// PlaceFuturesMarketOrder places a KuCoin futures MARKET order with reduceOnly flag support.
func (k *KucoinConnector) PlaceFuturesMarketOrder(symbol, side string, size int64, reduceOnly bool) (*kucoinAPIResponse, error) {
	return k.PlaceFuturesOrder(symbol, side, "market", size, reduceOnly)
}

// ExecuteFuturesOrder sends a futures order to KuCoin without changing leverage.
func (k *KucoinConnector) ExecuteFuturesOrder(
	symbol string,
	side string, // "buy" or "sell"
	orderType string, // "limit" or "market"
	size int64,
	price *float64, // nil for market
	leverage string,
	reduceOnly bool,
) (map[string]interface{}, error) {

	clientOid := fmt.Sprintf("go-%d", time.Now().UnixNano())

	body := map[string]interface{}{
		"clientOid":  clientOid,
		"symbol":     symbol,
		"side":       side,
		"type":       orderType,
		"size":       size,
		"leverage":   leverage,
		"reduceOnly": reduceOnly,
	}

	if orderType == "limit" && price != nil {
		body["price"] = fmt.Sprintf("%f", *price)
	}

	b, err := json.Marshal(body)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"side":   side,
			"type":   orderType,
			"size":   size,
			"error":  err,
		}).Error("Failed to marshal futures order body")
		return nil, fmt.Errorf("marshal order body: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol":     symbol,
		"side":       side,
		"type":       orderType,
		"size":       size,
		"reduceOnly": reduceOnly,
		"body":       string(b),
	}).Info("Placing KuCoin futures order")

	resp, err := k.futuresClient.doRequest(
		http.MethodPost,
		"/api/v1/orders",
		"",
		string(b),
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"side":   side,
			"type":   orderType,
			"size":   size,
			"error":  err,
		}).Error("Failed to place KuCoin futures order")
		return nil, fmt.Errorf("place futures order: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol": symbol,
		"raw":    string(resp.Data),
	}).Debug("KuCoin futures order raw response")

	var out map[string]interface{}
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"raw":    string(resp.Data),
			"error":  err,
		}).Error("Failed to unmarshal KuCoin futures order response")
		return nil, fmt.Errorf("unmarshal order response: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol":    symbol,
		"clientOid": clientOid,
	}).Info("KuCoin futures order placed successfully")

	return out, nil
}

// CloseAllPositions is a placeholder to align KuCoin connector behavior with Phemex flows.
func (k *KucoinConnector) CloseAllPositions(symbol string) error {
	logger.WithField("symbol", symbol).Warn("CloseAllPositions for KuCoin is not implemented; skipping")
	return nil
}

// SetFuturesLeverage sets the leverage for a given futures symbol.
func (k *KucoinConnector) SetFuturesLeverage(symbol string, leverage int) error {
	body := map[string]interface{}{
		"symbol":   symbol,
		"leverage": leverage,
	}

	b, err := json.Marshal(body)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol":   symbol,
			"leverage": leverage,
			"error":    err,
		}).Error("Failed to marshal leverage body")
		return fmt.Errorf("marshal leverage body: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol":   symbol,
		"leverage": leverage,
		"body":     string(b),
	}).Info("Setting KuCoin futures leverage")

	_, err = k.futuresClient.doRequest(
		http.MethodPost,
		"/api/v1/position/leverage",
		"",
		string(b),
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol":   symbol,
			"leverage": leverage,
			"error":    err,
		}).Error("Failed to set KuCoin futures leverage")
		return fmt.Errorf("set leverage failed: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol":   symbol,
		"leverage": leverage,
	}).Info("KuCoin futures leverage set successfully")

	return nil
}

// ExecuteFuturesOrderLeverage sets leverage for the symbol and then sends a futures order.
func (k *KucoinConnector) ExecuteFuturesOrderLeverage(
	symbol string,
	side string, // "buy" or "sell"
	orderType string, // "limit" or "market"
	size int64,
	price *float64, // nil for market
	leverage int,
	reduceOnly bool,
) (map[string]interface{}, error) {

	logger.WithFields(logger.Fields{
		"symbol":   symbol,
		"leverage": leverage,
	}).Info("Executing KuCoin futures order with leverage")

	// 1) Set leverage first (if a positive leverage was provided)
	if leverage > 0 {
		if err := k.SetFuturesLeverage(symbol, leverage); err != nil {
			logger.WithFields(logger.Fields{
				"symbol":   symbol,
				"leverage": leverage,
				"error":    err,
			}).Error("Failed to set leverage before placing order")
			return nil, err
		}
	} else {
		logger.WithField("symbol", symbol).Info("Skipping leverage setup for KuCoin futures order")
	}

	// 2) Generate client OID
	clientOid := fmt.Sprintf("go-%d", time.Now().UnixNano())

	// 3) Build order body (no leverage in body)
	body := map[string]interface{}{
		"clientOid":  clientOid,
		"symbol":     symbol,
		"side":       side,
		"type":       orderType,
		"size":       size,
		"reduceOnly": reduceOnly,
	}

	if orderType == "limit" && price != nil {
		body["price"] = fmt.Sprintf("%f", *price)
	}

	b, err := json.Marshal(body)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol":   symbol,
			"side":     side,
			"type":     orderType,
			"size":     size,
			"leverage": leverage,
			"error":    err,
		}).Error("Failed to marshal leveraged futures order body")
		return nil, fmt.Errorf("marshal order body: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol":   symbol,
		"side":     side,
		"type":     orderType,
		"size":     size,
		"leverage": leverage,
		"body":     string(b),
	}).Info("Placing KuCoin leveraged futures order")

	resp, err := k.futuresClient.doRequest(
		http.MethodPost,
		"/api/v1/orders",
		"",
		string(b),
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol":   symbol,
			"side":     side,
			"type":     orderType,
			"size":     size,
			"leverage": leverage,
			"error":    err,
		}).Error("Failed to place KuCoin leveraged futures order")
		return nil, fmt.Errorf("place futures order: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol": symbol,
		"raw":    string(resp.Data),
	}).Debug("KuCoin leveraged futures order raw response")

	var out map[string]interface{}
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"raw":    string(resp.Data),
			"error":  err,
		}).Error("Failed to unmarshal KuCoin leveraged futures order response")
		return nil, fmt.Errorf("unmarshal order response: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol":    symbol,
		"clientOid": clientOid,
		"leverage":  leverage,
	}).Info("KuCoin leveraged futures order placed successfully")

	return out, nil
}

// GetFuturesTicker returns the raw KuCoin Futures ticker for a given symbol.
// Example: symbol = "XBTUSDTM"
func (k *KucoinConnector) GetFuturesTicker(symbol string) (map[string]interface{}, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	// Build query and full request path
	endpoint := "/api/v1/ticker"
	query := "symbol=" + symbol
	fullURL := k.futuresClient.baseURL + endpoint + "?" + query

	// Log the full URL that will be called
	logger.WithFields(map[string]interface{}{
		"method": http.MethodGet,
		"url":    fullURL,
		"symbol": symbol,
	}).Info("Calling KuCoin Futures Ticker")

	// Execute request
	resp, err := k.futuresClient.doRequest(
		http.MethodGet,
		endpoint,
		query,
		"",
	)
	if err != nil {
		// Log error with full context
		logger.WithFields(map[string]interface{}{
			"method": http.MethodGet,
			"url":    fullURL,
			"symbol": symbol,
			"error":  err,
		}).Error("KuCoin Futures Ticker request failed")

		return nil, fmt.Errorf("get futures ticker: %w", err)
	}

	// Debug raw response data (optional but useful)
	logger.WithFields(map[string]interface{}{
		"symbol": symbol,
		"raw":    string(resp.Data),
	}).Debug("KuCoin Futures Ticker raw response")

	// Unmarshal JSON response
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		logger.WithFields(map[string]interface{}{
			"symbol": symbol,
			"raw":    string(resp.Data),
			"error":  err,
		}).Error("Failed to unmarshal KuCoin Futures ticker response")

		return nil, fmt.Errorf("unmarshal futures ticker: %w", err)
	}

	// Success log
	logger.WithFields(map[string]interface{}{
		"symbol": symbol,
	}).Info("KuCoin Futures Ticker fetched successfully")

	return data, nil
}

// GetFuturesAvailableForSymbol returns the available USDT margin that can be used
// to open new positions for the given futures symbol (USDT-M).
//
// Note: For USDT-margined contracts, all symbols share the same USDT margin pool,
// so this method currently returns the global AvailableBalance for currency=USDT.
func (k *KucoinConnector) GetFuturesAvailableForSymbol(symbol string) (float64, error) {
	if symbol == "" {
		return 0, fmt.Errorf("symbol is required")
	}

	logger.WithField("symbol", symbol).Info("Fetching KuCoin futures available balance for symbol")

	resp, err := k.futuresClient.doRequest(
		http.MethodGet,
		"/api/v1/account-overview",
		"currency=USDT",
		"",
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"error":  err,
		}).Error("Failed to fetch KuCoin futures account overview")
		return 0, fmt.Errorf("fetch futures account overview: %w", err)
	}

	var fut kucoinFuturesAccountOverview
	if err := json.Unmarshal(resp.Data, &fut); err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"raw":    string(resp.Data),
			"error":  err,
		}).Error("Failed to unmarshal KuCoin futures account overview")
		return 0, fmt.Errorf("unmarshal futures account: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol":           symbol,
		"availableBalance": fut.AvailableBalance,
		"currency":         fut.Currency,
	}).Info("KuCoin futures available balance fetched")

	return fut.AvailableBalance, nil
}

// GetFuturesAvailableFromRiskUnit returns the futures available margin using the
// risk unit information. For KuCoin USDT-M contracts this currently delegates
// to GetFuturesAvailableForSymbol while keeping the explicit naming for risk
// calculations in the controller layer.
func (k *KucoinConnector) GetFuturesAvailableFromRiskUnit(symbol string) (float64, error) {
	return k.GetFuturesAvailableForSymbol(symbol)
}

// GetFuturesContractInfo fetches futures contract details for a specific symbol.
// Example: symbol = "XBTUSDTM"
func (k *KucoinConnector) GetFuturesContractInfo(symbol string) (*KucoinFuturesContract, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	endpoint := fmt.Sprintf("/api/v1/contracts/%s", symbol)
	logger.WithFields(logger.Fields{
		"symbol":   symbol,
		"endpoint": endpoint,
	}).Info("Fetching KuCoin futures contract info")

	resp, err := k.futuresClient.doRequest(
		http.MethodGet,
		endpoint,
		"",
		"",
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"error":  err,
		}).Error("Failed to fetch KuCoin futures contract info")
		return nil, fmt.Errorf("get futures contract info: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol": symbol,
		"raw":    string(resp.Data),
	}).Debug("KuCoin futures contract raw response")

	var contract KucoinFuturesContract
	if err := json.Unmarshal(resp.Data, &contract); err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"raw":    string(resp.Data),
			"error":  err,
		}).Error("Failed to unmarshal KuCoin futures contract info")
		return nil, fmt.Errorf("unmarshal futures contract info: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol": symbol,
	}).Info("KuCoin futures contract info fetched successfully")

	return &contract, nil
}

// GetFuturesContractInfoRaw returns the raw contract info as a map.
func (k *KucoinConnector) GetFuturesContractInfoRaw(symbol string) (map[string]interface{}, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	endpoint := fmt.Sprintf("/api/v1/contracts/%s", symbol)
	logger.WithFields(logger.Fields{
		"symbol":   symbol,
		"endpoint": endpoint,
	}).Info("Fetching KuCoin futures contract info (raw)")

	resp, err := k.futuresClient.doRequest(
		http.MethodGet,
		endpoint,
		"",
		"",
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"error":  err,
		}).Error("Failed to fetch KuCoin futures contract info (raw)")
		return nil, fmt.Errorf("get futures contract info: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol": symbol,
		"raw":    string(resp.Data),
	}).Debug("KuCoin futures contract raw response (raw)")

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		logger.WithFields(logger.Fields{
			"symbol": symbol,
			"raw":    string(resp.Data),
			"error":  err,
		}).Error("Failed to unmarshal KuCoin futures contract info (raw)")
		return nil, fmt.Errorf("unmarshal futures contract info: %w", err)
	}

	logger.WithFields(logger.Fields{
		"symbol": symbol,
	}).Info("KuCoin futures contract info (raw) fetched successfully")

	return data, nil
}

// ConvertUSDTToContracts converts a USDT amount (with leverage)
// into a contract size for a KuCoin futures symbol.
//
// Returns:
//   - size: integer number of contracts
//   - usdtUsed: effective USDT used after rounding
func (k *KucoinConnector) ConvertUSDTToContracts(
	symbol string,
	usdt float64,
	leverage int,
) (size int64, usdtUsed float64, err error) {

	logger.WithFields(logger.Fields{
		"symbol":   symbol,
		"usdt":     usdt,
		"leverage": leverage,
	}).Info("Converting USDT to KuCoin futures contracts")

	if usdt <= 0 {
		err = fmt.Errorf("usdt must be > 0, got %f", usdt)
		logger.WithError(err).Error("Invalid USDT amount for ConvertUSDTToContracts")
		return
	}
	if leverage <= 0 {
		err = fmt.Errorf("leverage must be > 0, got %d", leverage)
		logger.WithError(err).Error("Invalid leverage for ConvertUSDTToContracts")
		return
	}

	// 1) Get ticker to obtain the price
	ticker, err := k.GetFuturesTicker(symbol)
	if err != nil {
		err = fmt.Errorf("GetFuturesTicker failed: %w", err)
		logger.WithError(err).Error("Failed to get ticker in ConvertUSDTToContracts")
		return
	}

	price := toFloat(ticker["price"])
	if price <= 0 {
		price = toFloat(ticker["lastTradePrice"])
	}
	if price <= 0 {
		err = fmt.Errorf("invalid price for %s", symbol)
		logger.WithError(err).Error("Invalid price in ConvertUSDTToContracts")
		return
	}

	// 2) Get contract info to obtain the multiplier
	contract, err := k.GetFuturesContractInfo(symbol)
	if err != nil {
		err = fmt.Errorf("GetFuturesContractInfo failed: %w", err)
		logger.WithError(err).Error("Failed to get contract info in ConvertUSDTToContracts")
		return
	}

	multiplier := contract.Multiplier
	if multiplier <= 0 {
		err = fmt.Errorf("invalid contract multiplier for %s: %f", symbol, multiplier)
		logger.WithError(err).Error("Invalid multiplier in ConvertUSDTToContracts")
		return
	}

	contractsFloat := (usdt * float64(leverage)) / (price * multiplier)

	logger.WithFields(logger.Fields{
		"symbol":         symbol,
		"usdt":           usdt,
		"leverage":       leverage,
		"price":          price,
		"multiplier":     multiplier,
		"contractsFloat": contractsFloat,
	}).Debug("Intermediate values in ConvertUSDTToContracts")

	if contractsFloat <= 0 {
		err = fmt.Errorf("computed contracts <= 0 (usdt=%f, leverage=%d, price=%f, multiplier=%f)",
			usdt, leverage, price, multiplier)
		logger.WithError(err).Error("Computed contracts <= 0 in ConvertUSDTToContracts")
		return
	}

	size = int64(contractsFloat)
	if size == 0 && contractsFloat > 0 {
		size = 1
	}

	usdtUsed = float64(size) * price * multiplier / float64(leverage)

	logger.WithFields(logger.Fields{
		"symbol":   symbol,
		"size":     size,
		"usdtUsed": usdtUsed,
	}).Info("Converted USDT to KuCoin futures contracts successfully")

	return
}
