package model

import (
	"strategyexecutor/src/utils"
	"time"

	"github.com/shopspring/decimal"
)

type OHLCVResults struct {
	Symbol      string          `json:"symbol"`
	MaxPrice    decimal.Decimal `json:"max_price"`
	MinPrice    decimal.Decimal `json:"min_price"`
	FirstPrice  decimal.Decimal `json:"first_price"`
	LastPrice   decimal.Decimal `json:"last_price"`
	SumQuantity decimal.Decimal `json:"sum_quantity"`
	RoundedHour time.Time       `json:"rounded_hour"`
}

type OHLCVBase struct {
	ID       uint            `json:"id"`
	Datetime time.Time       `json:"datetime"`
	Open     decimal.Decimal `json:"open"`
	High     decimal.Decimal `json:"high"`
	Low      decimal.Decimal `json:"low"`
	Close    decimal.Decimal `json:"close"`
	Volume   decimal.Decimal `json:"volume"`
	Symbol   string          `json:"symbol"`
}

func (o *OHLCVBase) ConvertToOHLCVCrypto1h() *OHLCVCrypto1h {
	return &OHLCVCrypto1h{
		ID:       o.ID,
		Datetime: utils.ResetTime(o.Datetime, "hour"),
		Open:     o.Open,
		High:     o.High,
		Low:      o.Low,
		Close:    o.Close,
		Volume:   o.Volume,
		Symbol:   o.Symbol,
	}
}

func (o *OHLCVBase) ConvertToOHLCVCrypto1m() *OHLCVCrypto1m {
	return &OHLCVCrypto1m{
		ID:       o.ID,
		Datetime: utils.ResetTime(o.Datetime, "minute"),
		Open:     o.Open,
		High:     o.High,
		Low:      o.Low,
		Close:    o.Close,
		Volume:   o.Volume,
		Symbol:   o.Symbol,
	}
}
