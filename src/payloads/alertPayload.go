package payloads

type AlertPayload struct {
	Ticker                 string `json:"ticker"`
	Action                 string `json:"action"`
	Sentiment              string `json:"sentiment"`
	Quantity               string `json:"quantity"`
	Qty                    string `json:"qty"`
	Price                  string `json:"price"`
	Time                   string `json:"time"`
	Interval               string `json:"interval"`
	MarketPosition         string `json:"marketPosition"`
	PrevMarketPosition     string `json:"prevMarketPosition"`
	MarketPositionSize     string `json:"marketPositionSize"`
	PrevMarketPositionSize string `json:"prevMarketPositionSize"`
	AlertTime              string `json:"timestamp"`
}
