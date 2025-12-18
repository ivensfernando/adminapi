package externalmodel

import "time"

type TradingSignal struct {
	ID                     uint       `gorm:"primaryKey;column:id" json:"id"`
	OrderID                string     `gorm:"column:order_id" json:"order_id"`
	ExchangeName           string     `gorm:"column:exchange_name" json:"exchange_name"`
	Symbol                 string     `gorm:"column:symbol" json:"symbol"`
	Action                 string     `gorm:"column:action" json:"action"`
	OrderType              string     `gorm:"column:order_type" json:"order_type"`
	Qty                    float64    `gorm:"column:qty" json:"qty"`
	Price                  *float64   `gorm:"column:price" json:"price,omitempty"`
	MarketPosition         string     `gorm:"column:market_position" json:"market_position"`
	PrevMarketPosition     string     `gorm:"column:prev_market_position" json:"prev_market_position"`
	MarketPositionSize     float64    `gorm:"column:market_position_size" json:"market_position_size"`
	PrevMarketPositionSize float64    `gorm:"column:prev_market_position_size" json:"prev_market_position_size"`
	SignalToken            string     `gorm:"column:signal_token" json:"signal_token"`
	TimestampRaw           string     `gorm:"column:timestamp_raw" json:"timestamp_raw"`
	TimestampDT            *time.Time `gorm:"column:timestamp_dt" json:"timestamp_dt,omitempty"`
	Comment                string     `gorm:"column:comment" json:"comment"`
	Message                string     `gorm:"column:message" json:"message"`
	ReceivedAt             *time.Time `gorm:"column:received_at" json:"received_at,omitempty"`
}

// TableName Ensures that GORM uses the exact table name from the database.
func (TradingSignal) TableName() string {
	return "trade_tradingsignal"
}
