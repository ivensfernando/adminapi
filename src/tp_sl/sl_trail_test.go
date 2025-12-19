package tp_sl

import (
	"strategyexecutor/src/model"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func c(dt time.Time, o, h, l, cl string) model.OHLCVCrypto1m {
	return model.OHLCVCrypto1m{
		Symbol:   "BTCUSDT",
		Datetime: dt,
		Open:     d(o),
		High:     d(h),
		Low:      d(l),
		Close:    d(cl),
		Volume:   d("1"),
	}
}

func TestComputeNextStopLoss_NotEnoughCandles(t *testing.T) {
	now := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now, "100", "101", "99", "100"),
	}

	sl, raised := ComputeNextStopLossDirectional(SideLong, d("95"), candles, 20)
	if raised {
		t.Fatalf("expected raised=false")
	}
	if !sl.Equal(d("95")) {
		t.Fatalf("expected sl unchanged, got=%s", sl.String())
	}
}

func TestComputeNextStopLoss_PrevNotBullish_NoRaise(t *testing.T) {
	// prev candle is bearish: close <= open
	now := time.Date(2025, 3, 1, 0, 2, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-2*time.Minute), "100", "101", "99", "100"),
		c(now.Add(-1*time.Minute), "105", "106", "100", "104"), // prev: bearish (104 < 105)
		c(now, "106", "107", "103", "106"),
	}

	sl, raised := ComputeNextStopLossDirectional(SideLong, d("98"), candles, 3)
	if raised {
		t.Fatalf("expected raised=false")
	}
	if !sl.Equal(d("98")) {
		t.Fatalf("expected sl unchanged, got=%s", sl.String())
	}
}

func TestComputeNextStopLoss_RaiseToFloorAvg_ClampedToPrevLow(t *testing.T) {
	// prev candle bullish, floorAvg > prev.Low so we clamp down to prev.Low
	// lows (lookback 3) = 100, 101, 102 => avg = 101
	// prev.Low = 100.50 so candidate becomes 100.50
	now := time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-3*time.Minute), "110", "111", "100", "110"),
		c(now.Add(-2*time.Minute), "110", "112", "101", "111"),
		c(now.Add(-1*time.Minute), "100", "130", "100.50", "120"), // prev bullish
		c(now, "120", "121", "119", "120"),
	}

	currentSL := d("99.0")
	sl, raised := ComputeNextStopLossDirectional(SideLong, currentSL, candles, 3)

	if !raised {
		t.Fatalf("expected raised=true")
	}
	if !sl.Equal(d("100.50")) {
		t.Fatalf("expected sl=100.50 (clamped to prev low), got=%s", sl.String())
	}
}

func TestComputeNextStopLoss_RaiseToFloorAvg_WhenBelowPrevLow(t *testing.T) {
	// prev candle bullish, floorAvg < prev.Low so candidate is floorAvg
	// lows (lookback 3) = 100, 100, 100 => avg = 100
	// prev.Low = 110 so candidate = 100
	now := time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-3*time.Minute), "110", "111", "100", "110"),
		c(now.Add(-2*time.Minute), "110", "112", "100", "111"),
		c(now.Add(-1*time.Minute), "120", "130", "110", "125"), // prev bullish
		c(now, "125", "126", "124", "125"),
	}

	currentSL := d("95")
	sl, raised := ComputeNextStopLossDirectional(SideLong, currentSL, candles, 3)

	if !raised {
		t.Fatalf("expected raised=true")
	}
	if !sl.Equal(d("110")) {
		t.Fatalf("expected sl=110, got=%s", sl.String())
	}
}

func TestComputeNextStopLoss_NeverLowersStop(t *testing.T) {
	// candidate ends up <= currentSL, must not reduce
	now := time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-3*time.Minute), "110", "111", "100", "110"),
		c(now.Add(-2*time.Minute), "110", "112", "100", "111"),
		c(now.Add(-1*time.Minute), "120", "130", "110", "125"), // prev bullish
		c(now, "125", "126", "124", "125"),
	}

	// floorAvg (100,100,110) = 103.333..., prev.Low=110 => candidate=103.333...
	// currentSL is higher already
	currentSL := d("110")
	sl, raised := ComputeNextStopLossDirectional(SideLong, currentSL, candles, 3)

	if raised {
		t.Fatalf("expected raised=false, sl must not decrease")
	}
	if !sl.Equal(currentSL) {
		t.Fatalf("expected sl unchanged=%s got=%s", currentSL.String(), sl.String())
	}
}

func TestComputeNextStopLoss_LookbackLargerThanCandles_UsesAll(t *testing.T) {
	// lookback=50 but only 4 candles available
	now := time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-3*time.Minute), "100", "101", "90", "100"),
		c(now.Add(-2*time.Minute), "100", "101", "92", "101"),
		c(now.Add(-1*time.Minute), "100", "105", "94", "104"), // prev bullish
		c(now, "104", "106", "103", "105"),
	}

	// avg lows across all 4 = (90+92+94+103)/4 = 94.75
	// prev.Low = 94 so clamp to 94
	currentSL := d("80")
	sl, raised := ComputeNextStopLossDirectional(SideLong, currentSL, candles, 50)

	if !raised {
		t.Fatalf("expected raised=true")
	}
	if !sl.Equal(d("94")) {
		t.Fatalf("expected sl=94 (clamped), got=%s", sl.String())
	}
}

func TestIsBullish(t *testing.T) {
	now := time.Now()
	if !IsBullish(c(now, "100", "101", "99", "100.01")) {
		t.Fatalf("expected bullish")
	}
	if IsBullish(c(now, "100", "101", "99", "100")) {
		t.Fatalf("expected not bullish when close==open")
	}
	if IsBullish(c(now, "100", "101", "99", "99")) {
		t.Fatalf("expected not bullish")
	}
}

func TestAvgLow(t *testing.T) {
	now := time.Now()
	window := []model.OHLCVCrypto1m{
		c(now, "0", "0", "10", "0"),
		c(now, "0", "0", "20", "0"),
		c(now, "0", "0", "30", "0"),
	}
	avg := AvgLow(window)
	if !avg.Equal(d("20")) {
		t.Fatalf("expected avg=20 got=%s", avg.String())
	}
	if !AvgLow(nil).Equal(decimal.Zero) {
		t.Fatalf("expected avg=0 for empty slice")
	}
}

func TestComputeNextStopLoss_Short_NotEnoughCandles(t *testing.T) {
	now := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now, "100", "101", "99", "100"),
	}

	sl, moved := ComputeNextStopLossDirectional(SideShort, d("110"), candles, 20)
	if moved {
		t.Fatalf("expected moved=false")
	}
	if !sl.Equal(d("110")) {
		t.Fatalf("expected sl unchanged, got=%s", sl.String())
	}
}

func TestComputeNextStopLoss_Short_PrevNotBearish_NoMove(t *testing.T) {
	now := time.Date(2025, 3, 1, 0, 2, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-2*time.Minute), "100", "105", "99", "100"),
		c(now.Add(-1*time.Minute), "100", "106", "98", "101"), // prev bullish, not bearish
		c(now, "101", "107", "97", "100"),
	}

	sl, moved := ComputeNextStopLossDirectional(SideShort, d("120"), candles, 3)
	if moved {
		t.Fatalf("expected moved=false")
	}
	if !sl.Equal(d("120")) {
		t.Fatalf("expected sl unchanged, got=%s", sl.String())
	}
}

func TestComputeNextStopLoss_Short_LowerToCeilAvg_ClampedToPrevHigh(t *testing.T) {
	// prev candle bearish
	// ceilAvg < prev.High, so we clamp up to prev.High
	// then since prev.High < currentSL, we tighten down to prev.High
	now := time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-3*time.Minute), "100", "120", "90", "100"),
		c(now.Add(-2*time.Minute), "100", "121", "90", "100"),
		c(now.Add(-1*time.Minute), "130", "140", "120", "125"), // prev bearish (125 < 130), prev.High=140
		c(now, "125", "126", "110", "124"),
	}

	// window highs for lookback=3 are: 121, 140, 126 => avg = 129
	// clamp => candidate = max(129, 140) = 140
	currentSL := d("145")
	sl, moved := ComputeNextStopLossDirectional(SideShort, currentSL, candles, 3)

	if !moved {
		t.Fatalf("expected moved=true")
	}
	if !sl.Equal(d("140")) {
		t.Fatalf("expected sl=140 (clamped to prev high), got=%s", sl.String())
	}
}

func TestComputeNextStopLoss_Short_LowerToCeilAvg_WhenAbovePrevHigh(t *testing.T) {
	// prev candle bearish
	// ceilAvg > prev.High, so candidate is ceilAvg
	// if candidate < currentSL, tighten down to candidate
	now := time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-3*time.Minute), "100", "150", "90", "100"),
		c(now.Add(-2*time.Minute), "100", "150", "90", "100"),
		c(now.Add(-1*time.Minute), "130", "110", "120", "120"), // prev bearish (120 < 130), prev.High=110
		c(now, "120", "150", "110", "119"),
	}

	// window highs for lookback=3 are: 150, 110, 150 => avg = 136.666...
	// candidate = max(136.666..., 110) = 136.666...
	currentSL := d("140")
	sl, moved := ComputeNextStopLossDirectional(SideShort, currentSL, candles, 3)

	if !moved {
		t.Fatalf("expected moved=true")
	}

	expected := d("410").Div(d("3")) // 136.666...
	if !sl.Equal(expected) {
		t.Fatalf("expected sl=%s got=%s", expected.String(), sl.String())
	}
}

func TestComputeNextStopLoss_Short_NeverRaisesStop(t *testing.T) {
	// If candidate >= currentSL, we must not move the SL up for shorts
	now := time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-3*time.Minute), "100", "150", "90", "100"),
		c(now.Add(-2*time.Minute), "100", "150", "90", "100"),
		c(now.Add(-1*time.Minute), "130", "140", "120", "120"), // prev bearish, prev.High=140
		c(now, "120", "150", "110", "119"),
	}

	// window highs for lookback=3 are: 150, 140, 150 => avg = 146.666...
	// candidate = max(146.666..., 140) = 146.666...
	currentSL := d("140") // current is below candidate, so do not move
	sl, moved := ComputeNextStopLossDirectional(SideShort, currentSL, candles, 3)

	if moved {
		t.Fatalf("expected moved=false")
	}
	if !sl.Equal(currentSL) {
		t.Fatalf("expected sl unchanged=%s got=%s", currentSL.String(), sl.String())
	}
}

func TestComputeNextStopLoss_Short_LookbackLargerThanCandles_UsesAll(t *testing.T) {
	now := time.Date(2025, 3, 1, 0, 3, 0, 0, time.UTC)
	candles := []model.OHLCVCrypto1m{
		c(now.Add(-3*time.Minute), "100", "110", "90", "100"),
		c(now.Add(-2*time.Minute), "100", "120", "90", "100"),
		c(now.Add(-1*time.Minute), "130", "140", "120", "120"), // prev bearish, prev.High=140
		c(now, "120", "130", "110", "119"),
	}

	// lookback=50 => uses all 4 highs: 110,120,140,130 => avg=125
	// candidate = max(125, 140) = 140
	currentSL := d("150")
	sl, moved := ComputeNextStopLossDirectional(SideShort, currentSL, candles, 50)

	if !moved {
		t.Fatalf("expected moved=true")
	}
	if !sl.Equal(d("140")) {
		t.Fatalf("expected sl=140 (clamped), got=%s", sl.String())
	}
}
