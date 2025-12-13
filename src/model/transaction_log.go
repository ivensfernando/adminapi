package model

import "time"

type TransactionLog struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	StrategyID       *uint          `gorm:"index" json:"strategy_id,omitempty"`
	StrategyActionID *uint          `gorm:"index" json:"strategy_action_id,omitempty"`
	OrderID          *uint          `gorm:"index" json:"order_id,omitempty"`
	Level            string         `gorm:"size:20;not null" json:"level"`
	Message          string         `gorm:"size:1024;not null" json:"message"`
	Metadata         map[string]any `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}
