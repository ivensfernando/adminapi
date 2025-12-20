package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type UserExchange struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"not null;index:idx_user_exchange,unique" json:"user_id"`
	// LegacyUserID keeps the previous identifier used before the User model existed.
	// It remains available for backward compatibility but is no longer used as a key.
	LegacyUserID      string    `gorm:"size:60;column:legacy_user_id" json:"legacy_user_id,omitempty"`
	ExchangeID        uint      `gorm:"not null;index:idx_user_exchange,unique" json:"exchange_id"`
	APIKeyHash        string    `gorm:"column:api_key;type:text" json:"-"`
	APISecretHash     string    `gorm:"column:api_secret;type:text" json:"-"`
	APIPassphraseHash string    `gorm:"column:api_passphrase;type:text" json:"-"`
	OrderSizePercent  int       `gorm:"column:order_size_percent" json:"order_size_percent"`
	RunOnServer       bool      `gorm:"column:run_on_server" json:"run_on_server"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	WeekendHolidayMultiplier  decimal.Decimal `gorm:"column:weekend_holiday_multiplier" json:"weekend_holiday_multiplier"`
	DeadZoneMultiplier        decimal.Decimal `gorm:"column:dead_zone_multiplier" json:"dead_zone_multiplier"`
	AsiaMultiplier            decimal.Decimal `gorm:"column:asia_multiplier" json:"asia_multiplier"`
	LondonMultiplier          decimal.Decimal `gorm:"column:london_multiplier" json:"london_multiplier"`
	USMultiplier              decimal.Decimal `gorm:"column:us_multiplier" json:"us_multiplier"`
	DefaultMultiplier         decimal.Decimal `gorm:"column:default_multiplier" json:"default_multiplier"`
	EnableNoTradeWindow       bool            `gorm:"column:enable_no_trade_window" json:"enable_no_trade_window"`
	NoTradeWindowOrdersClosed bool            `gorm:"column:no_trade_window_orders_closed" json:"no_trade_window_orders_closed"`

	Exchange *Exchange `gorm:"constraint:OnDelete:CASCADE" json:"exchange"`
}
