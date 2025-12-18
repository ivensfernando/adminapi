package controller

import "testing"

func TestNormalizeToKucoinFuturesSymbol(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"BTCUSD", "XBTUSDTM"},
		{"ethusdt", "ETHUSDTM"},
		{"XBTUSDTM", "XBTUSDTM"},
	}

	for _, tt := range tests {
		if got := NormalizeToKucoinFuturesSymbol(tt.input); got != tt.expected {
			t.Fatalf("expected %s -> %s, got %s", tt.input, tt.expected, got)
		}
	}
}

func TestPercentOfFloatSafe(t *testing.T) {
	if got := PercentOfFloatSafe(200, 10); got != 20 {
		t.Fatalf("expected 10%% of 200 to be 20, got %f", got)
	}

	if got := PercentOfFloatSafe(100, 0); got != 1 {
		t.Fatalf("percent should clamp to minimum, expected 1 got %f", got)
	}

	if got := PercentOfFloatSafe(100, 150); got != 100 {
		t.Fatalf("percent should clamp to maximum, expected 100 got %f", got)
	}
}
