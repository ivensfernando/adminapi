package model

import (
	"fmt"
	"time"
)

//type Trade struct {
//	ID        uint      `gorm:"primaryKey" json:"id"`
//	Symbol    string    `json:"symbol"`
//	Price     float64   `json:"price"`
//	Quantity  int       `json:"quantity"`
//	TradeDate time.Time `json:"trade_date"`
//
//	Type       string   `gorm:"not null" json:"type"`
//	Leverage   *float64 `json:"leverage,omitempty"`
//	EntryPrice float64  `json:"entry_price"`
//	ExitPrice  float64  `json:"exit_price"`
//	Fee        *float64 `json:"fee,omitempty"`
//	Indicators *string  `json:"indicators,omitempty"`
//	Sentiment  *string  `json:"sentiment,omitempty"`
//	StopLoss   *float64 `json:"stop_loss,omitempty"`
//	TakeProfit *float64 `json:"take_profit,omitempty"`
//	Exchange   *string  `json:"exchange,omitempty"`
//
//	UserID    uint `json:"user_id"`                                        // Foreign key
//	User      User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // Optional join
//	CreatedAt time.Time
//	UpdatedAt time.Time
//}
//
//type TradePayload struct {
//	Symbol     string   `json:"symbol"`
//	TradeDate  string   `json:"trade_date"` // keep as string
//	Type       string   `json:"type"`
//	Leverage   *float64 `json:"leverage"`
//	EntryPrice float64  `json:"entry_price"`
//	ExitPrice  float64  `json:"exit_price"`
//	Fee        *float64 `json:"fee"`
//	Indicators *string  `json:"indicators"`
//	Sentiment  *string  `json:"sentiment"`
//	StopLoss   *float64 `json:"stop_loss"`
//	TakeProfit *float64 `json:"take_profit"`
//	Exchange   *string  `json:"exchange"`
//}
//
//type TradeResponse struct {
//	ID         uint     `json:"id"`
//	Symbol     string   `json:"symbol"`
//	TradeDate  string   `json:"trade_date"` // keep as string
//	Type       string   `json:"type"`
//	Leverage   *float64 `json:"leverage"`
//	EntryPrice float64  `json:"entry_price"`
//	ExitPrice  float64  `json:"exit_price"`
//	Fee        *float64 `json:"fee"`
//	Indicators *string  `json:"indicators"`
//	Sentiment  *string  `json:"sentiment"`
//	StopLoss   *float64 `json:"stop_loss"`
//	TakeProfit *float64 `json:"take_profit"`
//	Exchange   *string  `json:"exchange"`
//}

func parseTradeDateTime(dateStr, timeStr string) (time.Time, error) {
	if dateStr == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("date or time missing")
	}

	// Assume "2025-07-11T21:16:05.574Z" + "12:00"
	parsedDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %v", err)
	}

	layout := "2006-01-02 15:04"
	combined := fmt.Sprintf("%s %s", parsedDate.Format("2006-01-02"), timeStr)
	return time.Parse(layout, combined)
}

type Trade struct {
	ID uint `gorm:"primaryKey" json:"id"`

	Exchange  *string   `json:"exchange,omitempty"`
	Symbol    string    `json:"symbol"`
	TradeDate time.Time `json:"trade_date"`
	TradeTime string    `json:"trade_time"`

	MarginMode        string   `json:"margin_mode"`
	Leverage          *float64 `json:"leverage,omitempty"`
	AssetMode         string   `json:"asset_mode"`
	OrderType         string   `json:"order_type"`
	Quantity          float64  `json:"quantity"`
	StopPrice         float64  `json:"stop_price"`
	Price             float64  `json:"price"`
	TakeProfitEnabled bool     `json:"is_take_profit"`
	ReduceOnly        bool     `json:"is_reduce_only"`
	StopLoss          *float64 `json:"stop_loss,omitempty"`
	TakeProfit        *float64 `json:"take_profit,omitempty"`
	IsShort           bool     `json:"is_short"`
	IsLong            bool     `json:"is_long"`

	Type         string   `gorm:"not null" json:"type"`
	ContractType *string  `json:"contract_type"`
	EntryPrice   float64  `json:"entry_price"`
	ExitPrice    float64  `json:"exit_price"`
	Fee          *float64 `json:"fee,omitempty"`
	Indicators   *string  `json:"indicators,omitempty"`
	Sentiment    *string  `json:"sentiment,omitempty"`

	Notes *string `json:"notes,omitempty"`

	UserID    uint `json:"user_id"`                                        // FK
	User      User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // Opcional
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TradePayload struct {
	//trade form
	Exchange  *string `json:"exchange"`
	Symbol    string  `json:"symbol"`
	TradeDate string  `json:"tradeDate"` // ex: "2025-07-11T21:16:05.574Z"
	TradeTime string  `json:"tradeTime"` // ex: "12:00"
	//binance form
	MarginMode        string   `json:"marginMode"`
	Leverage          *float64 `json:"leverage"`
	AssetMode         string   `json:"assetMode"`
	OrderType         string   `json:"orderType"`
	Price             float64  `json:"price"`
	Quantity          float64  `json:"size"` // `size` vindo do frontend
	StopPrice         float64  `json:"stopPrice"`
	TakeProfitEnabled bool     `json:"takeProfitEnabled"`
	ReduceOnly        bool     `json:"reduceOnly"`
	TakeProfit        *float64 `json:"takeProfit"`
	StopLoss          *float64 `json:"stopLoss"`
	IsShort           bool     `json:"isShort"`
	IsLong            bool     `json:"isLong"`
	// Xx form
	Notes        *string  `json:"notes"`
	ContractType *string  `json:"contractType"`
	Type         string   `json:"type"`
	EntryPrice   float64  `json:"entryPrice"`
	ExitPrice    float64  `json:"exitPrice"`
	Fee          *float64 `json:"fee"`
	Indicators   *string  `json:"indicators"`
	Sentiment    *string  `json:"sentiment"`
}

type UpdateTradePayload struct {
	Exchange          **string  `json:"exchange,omitempty"`
	Symbol            *string   `json:"symbol,omitempty"`
	TradeDate         *string   `json:"tradeDate,omitempty"`
	TradeTime         *string   `json:"tradeTime,omitempty"`
	MarginMode        *string   `json:"marginMode,omitempty"`
	Leverage          **float64 `json:"leverage,omitempty"`
	AssetMode         *string   `json:"assetMode,omitempty"`
	OrderType         *string   `json:"orderType,omitempty"`
	Price             *float64  `json:"price,omitempty"`
	Quantity          *float64  `json:"size,omitempty"`
	StopPrice         *float64  `json:"stopPrice,omitempty"`
	TakeProfitEnabled *bool     `json:"takeProfitEnabled,omitempty"`
	ReduceOnly        *bool     `json:"reduceOnly,omitempty"`
	TakeProfit        **float64 `json:"takeProfit,omitempty"`
	StopLoss          **float64 `json:"stopLoss,omitempty"`
	IsShort           *bool     `json:"isShort,omitempty"`
	IsLong            *bool     `json:"isLong,omitempty"`
	Notes             **string  `json:"notes,omitempty"`
	Type              *string   `json:"type,omitempty"`
	EntryPrice        *float64  `json:"entry_price,omitempty"`
	ExitPrice         *float64  `json:"exit_price,omitempty"`
	Fee               **float64 `json:"fee,omitempty"`
	Indicators        **string  `json:"indicators,omitempty"`
	Sentiment         **string  `json:"sentiment,omitempty"`
}

type TradeResponse struct {
	ID        uint    `json:"id"`
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Quantity  float64 `json:"quantity"`
	TradeDate string  `json:"trade_date"` // enviado como string ISO
	//tradeTime
	Leverage   *float64 `json:"leverage,omitempty"`
	TakeProfit *float64 `json:"take_profit,omitempty"`
	StopLoss   *float64 `json:"stop_loss,omitempty"`
	Exchange   *string  `json:"exchange,omitempty"`
	Notes      *string  `json:"notes,omitempty"`
	//ID         uint     `json:"id"`
	//	Symbol     string   `json:"symbol"`
	//	TradeDate  string   `json:"trade_date"` // keep as string
	Type string `json:"type"`
	//	Leverage   *float64 `json:"leverage"`
	EntryPrice float64  `json:"entry_price"`
	ExitPrice  float64  `json:"exit_price"`
	Fee        *float64 `json:"fee"`
	Indicators *string  `json:"indicators"`
	Sentiment  *string  `json:"sentiment"`
	//	StopLoss   *float64 `json:"stop_loss"`
	//	TakeProfit *float64 `json:"take_profit"`
	//	Exchange   *string  `json:"exchange"`
}
