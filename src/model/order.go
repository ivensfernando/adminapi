package model

import "time"

const (
	OrderStatusPending  = "pending"
	OrderStatusExecuted = "executed"
	OrderStatusFailed   = "failed"
)

type Order struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	StrategyActionID   uint       `gorm:"index" json:"strategy_action_id"`
	StrategyID         uint       `gorm:"index" json:"strategy_id"`
	UserID             uint       `gorm:"index" json:"user_id"`
	ExchangeID         uint       `gorm:"index" json:"exchange_id"`
	ExternalID         string     `gorm:"size:255" json:"external_id"`
	Symbol             string     `json:"symbol"`
	Side               string     `json:"side"`
	OrderType          string     `json:"order_type"`
	Quantity           float64    `json:"quantity"`
	Price              *float64   `json:"price,omitempty"`
	StopLossPct        float64    `json:"stop_loss_pct"`
	TakeProfitPct      float64    `json:"take_profit_pct"`
	Status             string     `gorm:"size:50;not null;default:pending" json:"status"`
	TriggeredByAlertID *uint      `json:"triggered_by_alert_id,omitempty"`
	ExecutedAt         *time.Time `json:"executed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	Action   *StrategyAction `gorm:"constraint:OnDelete:SET NULL" json:"action,omitempty"`
	Strategy *Strategy       `gorm:"constraint:OnDelete:SET NULL" json:"strategy,omitempty"`
}

type Position struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index" json:"user_id"`
	ExchangeID uint       `gorm:"index" json:"exchange_id"`
	Symbol     string     `json:"symbol"`
	Side       string     `json:"side"`
	Quantity   float64    `json:"quantity"`
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
