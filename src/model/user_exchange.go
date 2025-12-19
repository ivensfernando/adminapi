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

	WeekendHolidayMultiplier decimal.Decimal `json:"WeekendHolidayMultiplier"`
	DeadZoneMultiplier       decimal.Decimal `json:"DeadZoneMultiplier"`
	AsiaMultiplier           decimal.Decimal `json:"AsiaMultiplier"`
	LondonMultiplier         decimal.Decimal `json:"LondonMultiplier"`
	USMultiplier             decimal.Decimal `json:"USMultiplier"`
	DefaultMultiplier        decimal.Decimal `json:"DefaultMultiplier"`

	EnableNoTradeWindow bool `json:"EnableNoTradeWindow"`

	Exchange *Exchange `gorm:"constraint:OnDelete:CASCADE" json:"exchange"`
}
