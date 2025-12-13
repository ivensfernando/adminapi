package model

import "time"

type Position struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index" json:"user_id"`
	ExchangeID uint       `gorm:"index" json:"exchange_id"`
	Symbol     string     `json:"symbol"`
	Side       string     `json:"side"`
	Quantity   float64    `json:"quantity"`
	Pnl        float64    `json:"pnl"`
	EntryPrice float64    `json:"entry_price"`
	ExitPrice  *float64   `json:"exit_price,omitempty"`
	OpenedAt   time.Time  `json:"opened_at"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`
	Status     string     `gorm:"size:50;not null;default:open" json:"status"`
	OrderID    *uint      `gorm:"index" json:"order_id,omitempty"`
	StrategyID *uint      `gorm:"index" json:"strategy_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

const (
	PositionStatusOpen   = "open"
	PositionStatusClosed = "closed"
)
