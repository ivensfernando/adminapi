package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type OHLCVCrypto1m struct {
	ID       uint            `gorm:"primaryKey"`
	Symbol   string          `json:"symbol"   gorm:"type:varchar(50);not null;uniqueIndex:ux_ohlcv_crypto_1m_symbol_datetime,priority:1;index:idx_ohlcv_crypto_1m_symbol_datetime,priority:1"`
	Datetime time.Time       `json:"datetime" gorm:"not null;uniqueIndex:ux_ohlcv_crypto_1m_symbol_datetime,priority:2;index:idx_ohlcv_crypto_1m_symbol_datetime,priority:2;index:idx_ohlcv_crypto_1m_datetime"`
	Open     decimal.Decimal `json:"open"   gorm:"type:double precision;not null"`
	High     decimal.Decimal `json:"high"   gorm:"type:double precision;not null"`
	Low      decimal.Decimal `json:"low"    gorm:"type:double precision;not null"`
	Close    decimal.Decimal `json:"close"  gorm:"type:double precision;not null"`
	Volume   decimal.Decimal `json:"volume" gorm:"type:double precision;not null"`
}

func (OHLCVCrypto1m) TableName() string {
	return "ohlcv_crypto_1m"
}

func (o OHLCVCrypto1m) ConvertToOHLCVBase() *OHLCVBase {
	return &OHLCVBase{
		ID:       o.ID,
		Datetime: o.Datetime,
		Open:     o.Open,
		High:     o.High,
		Low:      o.Low,
		Close:    o.Close,
		Volume:   o.Volume,
		Symbol:   o.Symbol,
	}
}

type OHLCVCrypto1h struct {
	ID       uint            `gorm:"primaryKey"`
	Symbol   string          `json:"symbol"   gorm:"type:varchar(50);not null;uniqueIndex:ux_ohlcv_crypto_1h_symbol_datetime,priority:1;index:idx_ohlcv_crypto_1h_symbol_datetime,priority:1"`
	Datetime time.Time       `json:"datetime" gorm:"not null;uniqueIndex:ux_ohlcv_crypto_1h_symbol_datetime,priority:2;index:idx_ohlcv_crypto_1h_symbol_datetime,priority:2;index:idx_ohlcv_crypto_1h_datetime"`
	Open     decimal.Decimal `json:"open"   gorm:"type:double precision;not null"`
	High     decimal.Decimal `json:"high"   gorm:"type:double precision;not null"`
	Low      decimal.Decimal `json:"low"    gorm:"type:double precision;not null"`
	Close    decimal.Decimal `json:"close"  gorm:"type:double precision;not null"`
	Volume   decimal.Decimal `json:"volume" gorm:"type:double precision;not null"`
}

func (OHLCVCrypto1h) TableName() string {
	return "ohlcv_crypto_1h"
}

func (o OHLCVCrypto1h) ConvertToOHLCVBase() *OHLCVBase {
	return &OHLCVBase{
		ID:       o.ID,
		Datetime: o.Datetime,
		Open:     o.Open,
		High:     o.High,
		Low:      o.Low,
		Close:    o.Close,
		Volume:   o.Volume,
		Symbol:   o.Symbol,
	}
}
