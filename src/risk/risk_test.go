package risk

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func nyDate(year int, month time.Month, day, hour int) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		// fallback. still deterministic. hours will be interpreted as UTC
		return time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
	}
	return time.Date(year, month, day, hour, 0, 0, 0, loc)
}

func TestCalculateSizeByNYSession_WithNoTradeWindow(t *testing.T) {
	baseSize := decimal.NewFromFloat(1.0)

	cfg := SessionSizeConfig{
		WeekendHolidayMultiplier: decimal.RequireFromString("10"),
		DeadZoneMultiplier:       decimal.RequireFromString("20"),
		AsiaMultiplier:           decimal.RequireFromString("30"),
		LondonMultiplier:         decimal.RequireFromString("40"),
		USMultiplier:             decimal.RequireFromString("50"),
		DefaultMultiplier:        decimal.RequireFromString("60"),
		EnableNoTradeWindow:      true,
	}

	tests := []struct {
		name        string
		at          time.Time
		wantSession Session
		wantSize    decimal.Decimal
	}{
		{
			name:        "Asia session Tuesday 21.00 NY",
			at:          nyDate(2025, time.March, 4, 21),
			wantSession: SessionAsia,
			wantSize:    decimal.RequireFromString("30"),
		},
		{
			name:        "London session Tuesday 04.00 NY",
			at:          nyDate(2025, time.March, 4, 4),
			wantSession: SessionLondon,
			wantSize:    decimal.RequireFromString("40"),
		},
		{
			name:        "US session Tuesday 10.00 NY",
			at:          nyDate(2025, time.March, 4, 10),
			wantSession: SessionUS,
			wantSize:    decimal.RequireFromString("50"),
		},
		{
			name:        "Dead zone Tuesday 18.00 NY",
			at:          nyDate(2025, time.March, 4, 18),
			wantSession: SessionDeadZone,
			wantSize:    decimal.RequireFromString("20"),
		},
		{
			name:        "Friday before no trade window (08.00 NY. London session)",
			at:          nyDate(2025, time.March, 7, 8), // Friday 8.00
			wantSession: SessionLondon,
			wantSize:    decimal.RequireFromString("40"),
		},
		{
			name:        "Friday in no trade window (10.00 NY)",
			at:          nyDate(2025, time.March, 7, 10), // Friday 10.00
			wantSession: SessionNoTrade,
			wantSize:    decimal.Zero,
		},
		{
			name:        "Saturday always no trade",
			at:          nyDate(2025, time.March, 8, 12),
			wantSession: SessionNoTrade,
			wantSize:    decimal.Zero,
		},
		{
			name:        "Sunday in no trade window (01.00 NY)",
			at:          nyDate(2025, time.March, 9, 1),
			wantSession: SessionNoTrade,
			wantSize:    decimal.Zero,
		},
		{
			name:        "Sunday. no trade window (01.00 NY. Asia session)",
			at:          nyDate(2025, time.March, 9, 1),
			wantSession: SessionNoTrade,
			wantSize:    decimal.RequireFromString("0"),
		},
		{
			name:        "Sunday after no trade window (08.00 NY. UK session)",
			at:          nyDate(2025, time.March, 9, 3),
			wantSession: SessionLondon,
			wantSize:    decimal.RequireFromString("40"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSize, gotSession := CalculateSizeByNYSession(baseSize, tt.at, &cfg)

			if gotSession != tt.wantSession {
				t.Fatalf("session mismatch. got=%s want=%s", gotSession, tt.wantSession)
			}
			if !gotSize.Equal(tt.wantSize) {
				t.Fatalf("size mismatch. got=%s want=%s", gotSize.String(), tt.wantSize.String())
			}
		})
	}
}

func TestCalculateSizeByNYSession_WeekendWithoutNoTradeWindow(t *testing.T) {
	baseSize := decimal.NewFromFloat(1.0)

	cfg := SessionSizeConfig{
		WeekendHolidayMultiplier: decimal.RequireFromString("10"),
		DeadZoneMultiplier:       decimal.RequireFromString("20"),
		AsiaMultiplier:           decimal.RequireFromString("30"),
		LondonMultiplier:         decimal.RequireFromString("40"),
		USMultiplier:             decimal.RequireFromString("50"),
		DefaultMultiplier:        decimal.RequireFromString("60"),
		EnableNoTradeWindow:      false,
	}

	// Pick a Saturday that is not one of the holidays in isHoliday.
	at := nyDate(2025, time.March, 8, 12) // Saturday noon

	gotSize, gotSession := CalculateSizeByNYSession(baseSize, at, &cfg)

	if gotSession != SessionWeekendHoliday {
		t.Fatalf("session mismatch. got=%s want=%s", gotSession, SessionWeekendHoliday)
	}
	wantSize := decimal.RequireFromString("10")
	if !gotSize.Equal(wantSize) {
		t.Fatalf("size mismatch. got=%s want=%s", gotSize.String(), wantSize.String())
	}
}

func TestCalculateSizeByNYSession_BaseSizeZero(t *testing.T) {
	cfg := DefaultSessionSizeConfig()
	baseSize := decimal.Zero
	at := nyDate(2025, time.March, 4, 10)

	gotSize, gotSession := CalculateSizeByNYSession(baseSize, at, cfg)

	if !gotSize.Equal(decimal.Zero) {
		t.Fatalf("expected size zero for baseSize<=0. got=%s", gotSize.String())
	}
	if gotSession != SessionDefault {
		t.Fatalf("expected default session when baseSize<=0. got=%s", gotSession)
	}
}

func TestCalculateSizeByNYSession_HolidayWithNoTradeWindow(t *testing.T) {
	baseSize := decimal.NewFromFloat(1.0)

	cfg := SessionSizeConfig{
		WeekendHolidayMultiplier: decimal.RequireFromString("10"),
		DeadZoneMultiplier:       decimal.RequireFromString("20"),
		AsiaMultiplier:           decimal.RequireFromString("30"),
		LondonMultiplier:         decimal.RequireFromString("40"),
		USMultiplier:             decimal.RequireFromString("50"),
		DefaultMultiplier:        decimal.RequireFromString("60"),
		EnableNoTradeWindow:      true,
	}

	// July 4, 2025. US holiday.
	at := nyDate(2025, time.July, 4, 12)

	gotSize, gotSession := CalculateSizeByNYSession(baseSize, at, &cfg)

	if gotSession != SessionNoTrade {
		t.Fatalf("session mismatch on holiday. got=%s want=%s", gotSession, SessionNoTrade)
	}
	if !gotSize.Equal(decimal.Zero) {
		t.Fatalf("size mismatch on holiday. got=%s want=0", gotSize.String())
	}
}

func TestCalculateSizeByNYSession_HolidayWithoutNoTradeWindow(t *testing.T) {
	baseSize := decimal.NewFromFloat(1.0)

	cfg := SessionSizeConfig{
		WeekendHolidayMultiplier: decimal.RequireFromString("10"),
		DeadZoneMultiplier:       decimal.RequireFromString("20"),
		AsiaMultiplier:           decimal.RequireFromString("30"),
		LondonMultiplier:         decimal.RequireFromString("40"),
		USMultiplier:             decimal.RequireFromString("50"),
		DefaultMultiplier:        decimal.RequireFromString("60"),
		EnableNoTradeWindow:      false,
	}

	at := nyDate(2025, time.July, 4, 12)

	gotSize, gotSession := CalculateSizeByNYSession(baseSize, at, &cfg)

	if gotSession != SessionWeekendHoliday {
		t.Fatalf("session mismatch on holiday without window. got=%s want=%s", gotSession, SessionWeekendHoliday)
	}
	wantSize := decimal.RequireFromString("10")
	if !gotSize.Equal(wantSize) {
		t.Fatalf("size mismatch on holiday without window. got=%s want=%s", gotSize.String(), wantSize.String())
	}
}

func TestCalculateSizeSimple(t *testing.T) {
	baseSize := decimal.NewFromFloat(0.001)

	cfg := SessionSizeConfig{
		WeekendHolidayMultiplier: decimal.NewFromFloat(0.15),
		DeadZoneMultiplier:       decimal.NewFromFloat(0.15),
		AsiaMultiplier:           decimal.NewFromFloat(0.75),
		LondonMultiplier:         decimal.NewFromFloat(1.25),
		USMultiplier:             decimal.NewFromFloat(1.25),
		DefaultMultiplier:        decimal.NewFromFloat(0.15),
		EnableNoTradeWindow:      false,
	}

	at := nyDate(2025, time.December, 16, 12)

	gotSize, gotSession := CalculateSizeByNYSession(baseSize, at, &cfg)

	if gotSession != SessionUS {
		t.Fatalf("session mismatch. got=%s want=%s", gotSession, SessionWeekendHoliday)
	}
	wantSize := decimal.RequireFromString("0.00125")
	if !gotSize.Equal(wantSize) {
		t.Fatalf("size mismatch. got=%s want=%s", gotSize.String(), wantSize.String())
	}
}
