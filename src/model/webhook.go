package model

import "time"

type Webhook struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null" json:"user_id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	Style       string    `gorm:"size:100;not null" json:"style"`
	Type        string    `gorm:"size:100;not null" json:"type"`
	Tickers     string    `json:"tickers"`
	Token       string    `gorm:"size:255;uniqueIndex;not null" json:"token"`
	Active      bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WebhookAlert struct {
	ID                     uint       `gorm:"primaryKey" json:"id"`
	WebhookID              uint       `gorm:"index;not null" json:"webhook_id"`
	UserID                 uint       `gorm:"index;not null" json:"user_id"`
	Ticker                 string     `json:"ticker"`
	Action                 string     `json:"action"`
	Sentiment              string     `json:"sentiment,omitempty"`
	Quantity               float64    `json:"quantity"`
	Price                  float64    `json:"price"`
	Interval               string     `json:"interval,omitempty"`
	MarketPosition         string     `json:"marketPosition,omitempty"`
	PrevMarketPosition     string     `json:"prevMarketPosition,omitempty"`
	MarketPositionSize     float64    `json:"marketPositionSize,omitempty"`
	PrevMarketPositionSize float64    `json:"prevMarketPositionSize,omitempty"`
	Time                   *time.Time `json:"time,omitempty"`
	AlertTime              *time.Time `json:"alert_time,omitempty"`
	ReceivedAt             time.Time  `json:"received_at"`
}
type T struct {
	Ticker   string `json:"ticker"`
	Action   string `json:"action"`
	Price    string `json:"price"`
	Time     string `json:"time"`
	Interval string `json:"interval"`
}

//{
//"ticker": {{ticker}},
//"exchange": {{exchange}},
//"action": "buy",
//"type": "limit",
//"price": {{close}},
//"message": "entry",
//"long SL": {{plot("Long SL")}},
//"long TP": {{plot("Long TP")}},
//"passphrase": "abcdefg",
//"subaccount": "Testing",
//"chart_url" : "https://www.tradingview.com/chart/jbSLq0oe"
//}
