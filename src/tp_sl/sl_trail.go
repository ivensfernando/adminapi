package tp_sl

import (
	"strategyexecutor/src/model"

	"github.com/shopspring/decimal"
)

type Side string

const (
	SideLong  Side = "long"
	SideShort Side = "short"
)

func IsBullish(c model.OHLCVCrypto1m) bool { return c.Close.GreaterThan(c.Open) }
func IsBearish(c model.OHLCVCrypto1m) bool { return c.Close.LessThan(c.Open) }

func AvgLow(candles []model.OHLCVCrypto1m) decimal.Decimal {
	if len(candles) == 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, c := range candles {
		sum = sum.Add(c.Low)
	}
	return sum.Div(decimal.NewFromInt(int64(len(candles))))
}

func AvgHigh(candles []model.OHLCVCrypto1m) decimal.Decimal {
	if len(candles) == 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, c := range candles {
		sum = sum.Add(c.High)
	}
	return sum.Div(decimal.NewFromInt(int64(len(candles))))
}

// ComputeNextStopLossDirectional applies trailing SL for long or short.
//
// Long:
// - gate: previous candle bullish
// - floor: avg(low) over lookback
// - clamp: candidate <= prev.Low
// - update: SL = max(SL, candidate)
//
// Short:
// - gate: previous candle bearish
// - ceiling: avg(high) over lookback
// - clamp: candidate >= prev.High
// - update: SL = min(SL, candidate)
func ComputeNextStopLossDirectional(
	side Side,
	currentSL decimal.Decimal,
	candles []model.OHLCVCrypto1m,
	lookback int,
) (newSL decimal.Decimal, moved bool) {
	if len(candles) < 2 {
		return currentSL, false
	}
	if lookback <= 0 {
		lookback = 20
	}
	if lookback > len(candles) {
		lookback = len(candles)
	}

	prev := candles[len(candles)-2]
	window := candles[len(candles)-lookback:]

	switch side {
	case SideLong:
		if !IsBullish(prev) {
			return currentSL, false
		}
		floorAvg := AvgLow(window)

		candidate := floorAvg
		if candidate.GreaterThan(prev.Low) {
			candidate = prev.Low
		}

		if candidate.GreaterThan(currentSL) {
			return candidate, true
		}
		return currentSL, false

	case SideShort:
		if !IsBearish(prev) {
			return currentSL, false
		}
		ceilAvg := AvgHigh(window)

		candidate := ceilAvg
		// For shorts, do not set stop below the last bearish candle high
		if candidate.LessThan(prev.High) {
			candidate = prev.High
		}

		// Stop only moves down for shorts
		if candidate.LessThan(currentSL) {
			return candidate, true
		}
		return currentSL, false

	default:
		return currentSL, false
	}
}
