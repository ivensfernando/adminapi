package model

import "time"

type Strategy struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint           `gorm:"not null;index" json:"user_id"`
	WebhookID      uint           `gorm:"not null;index" json:"webhook_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Description    string         `gorm:"size:512" json:"description"`
	Active         bool           `gorm:"not null;default:true" json:"active"`
	ApplyOnAlert   bool           `gorm:"not null;default:true" json:"apply_on_alert"`
	DryRun         bool           `gorm:"not null;default:false" json:"dry_run"`
	AlertFilters   map[string]any `gorm:"type:jsonb" json:"alert_filters,omitempty"`
	MaxConcurrent  uint           `gorm:"default:1" json:"max_concurrent"`
	ClosePrevious  bool           `gorm:"not null;default:false" json:"close_previous"`
	AllowHedge     bool           `gorm:"not null;default:false" json:"allow_hedge"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	LastExecutedAt *time.Time     `json:"last_executed_at"`

	Actions []StrategyAction `gorm:"foreignKey:StrategyID" json:"actions,omitempty"`
	User    *User            `gorm:"constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Webhook *Webhook         `gorm:"constraint:OnDelete:CASCADE" json:"webhook,omitempty"`
}

// StrategyAction defines how a strategy will react when triggered by an alert.
type StrategyAction struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	StrategyID     uint      `gorm:"not null;index" json:"strategy_id"`
	ExchangeID     uint      `gorm:"not null" json:"exchange_id"`
	UserExchangeID *uint     `gorm:"index" json:"user_exchange_id,omitempty"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	ActionType     string    `gorm:"size:50;not null" json:"action_type"`
	OrderType      string    `gorm:"size:50;not null" json:"order_type"`
	MarketType     string    `gorm:"size:30;not null;default:spot" json:"market_type"`
	Symbol         string    `gorm:"size:50;not null" json:"symbol"`
	Quantity       float64   `gorm:"not null" json:"quantity"`
	Price          *float64  `json:"price,omitempty"`
	UseLastPrice   bool      `gorm:"not null;default:true" json:"use_last_price"`
	Slippage       *float64  `json:"slippage,omitempty"`
	MaxQuantity    float64   `gorm:"not null;default:0" json:"max_quantity"`
	StopLossPct    float64   `gorm:"not null" json:"stop_loss_pct"`
	TakeProfitPct  float64   `gorm:"not null" json:"take_profit_pct"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Strategy *Strategy `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Exchange *Exchange `json:"exchange,omitempty"`
}
