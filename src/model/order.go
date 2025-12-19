package model

import "time"

const (
	OrderDirectionEntry = "entry"
	OrderDirectionExit  = "exit"
)

// Order represents an order that your system sends to the exchange.
type Order struct {
	ID uint `gorm:"primaryKey" json:"id"`
	//StrategyActionID *uint `gorm:"index" json:"strategy_action_id"`
	//StrategyID         *uint      `gorm:"index" json:"strategy_id"`
	UserID       uint   `gorm:"index" json:"user_id"`
	LegacyUserID string `gorm:"size:60;column:legacy_user_id" json:"legacy_user_id,omitempty"`
	ExchangeID   uint   `gorm:"index" json:"exchange_id"`
	//ExchangeResp  string   `json:"exchange_resp,omitempty"` dropped from db
	ExternalID    uint     `gorm:"index" json:"external_id"`
	Symbol        string   `json:"symbol"`
	Side          string   `json:"side"`
	PosSide       string   `json:"pos_side"`
	OrderType     string   `json:"order_type"`
	Quantity      float64  `json:"quantity"`
	Price         *float64 `json:"price,omitempty"`
	StopLossPct   float64  `json:"stop_loss_pct"`
	TakeProfitPct float64  `json:"take_profit_pct"`
	Status        string   `gorm:"size:50;not null;default:pending" json:"status"`
	OrderDir      string   `gorm:"size:10;not null;" json:"order_dir"` //entry , exit
	//TriggeredByAlertID *uint      `json:"triggered_by_alert_id,omitempty"`
	ExecutedAt *time.Time `json:"executed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Relationship fields

	// Belongs to StrategyAction (optional)
	//StrategyAction *StrategyAction `gorm:"constraint:OnDelete:SET NULL" json:"strategy_action,omitempty"`

	// Belongs to Strategy (optional)
	//Strategy *Strategy `gorm:"constraint:OnDelete:SET NULL" json:"strategy,omitempty"`

	// One-to-many relation: one order can have many execution logs
	Logs []OrderLog `gorm:"foreignKey:OrderID" json:"order_logs,omitempty"`
}

// TableName allows you to control the exact table name for orders.
func (Order) TableName() string {
	return "orders"
}
