package risk

import (
	"time"

	"github.com/shopspring/decimal"
)

// ----- session labels -----

type Session string

const (
	SessionWeekendHoliday Session = "weekend_holiday"
	SessionDeadZone       Session = "dead_zone"
	SessionAsia           Session = "asia_session"
	SessionLondon         Session = "london_session"
	SessionUS             Session = "us_session"
	SessionDefault        Session = "default"
	SessionNoTrade        Session = "no_trade"
	DaysPerWeek                   = 7
	OffsetDaysForNewYear          = 1
	NewYearDay                    = 1
	ThirdMondayOffset             = 2
	FourthThursdayOffset          = 3
)

// ----- config for multipliers -----

type SessionSizeConfig struct {
	WeekendHolidayMultiplier decimal.Decimal
	DeadZoneMultiplier       decimal.Decimal
	AsiaMultiplier           decimal.Decimal
	LondonMultiplier         decimal.Decimal
	USMultiplier             decimal.Decimal
	DefaultMultiplier        decimal.Decimal

	EnableNoTradeWindow bool
}

// DefaultSessionSizeConfig reasonable defaults, tweak as you like
func DefaultSessionSizeConfig() SessionSizeConfig {
	return SessionSizeConfig{
		WeekendHolidayMultiplier: decimal.NewFromFloat(0.15),
		DeadZoneMultiplier:       decimal.NewFromFloat(0.15),
		AsiaMultiplier:           decimal.NewFromFloat(0.75),
		LondonMultiplier:         decimal.NewFromFloat(1.0),
		USMultiplier:             decimal.NewFromFloat(1.25),
		DefaultMultiplier:        decimal.NewFromFloat(0.15),
		EnableNoTradeWindow:      true,
	}
}

// ----- public API -----

// CalculateSizeByNYSession baseSize. nominal size you want to trade (e.g. 0.001 BTC). now. current time, usually time.Now(). cfg. multipliers and flags.
// returns finalSize (possibly zero in no trade window) and the detected session.
func CalculateSizeByNYSession(
	baseSize decimal.Decimal,
	now time.Time,
	cfg SessionSizeConfig,
) (decimal.Decimal, Session) {
	if baseSize.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, SessionDefault
	}

	et := getEasternTime(now)

	// no trade window, NY based, derived from "Friday after UK session until Sunday begin UK session"
	if cfg.EnableNoTradeWindow && isNoTradeWindowNY(et) {
		return decimal.Zero, SessionNoTrade
	}

	sess := detectSession(et)
	mult := sizeMultiplierForSession(sess, cfg)

	return baseSize.Mul(mult), sess
}

// ----- helpers, using your original logic -----

func getEasternTime(t time.Time) time.Time {
	nyLocation, err := time.LoadLocation("America/New_York")
	if err != nil {
		return t.UTC()
	}
	return t.In(nyLocation)
}

// isNoTradeWindowNY "Friday after UK session and end Sunday begin UK session"
// UK session in your NY based functions is isLondonSession: 3 <= h < 9
// So, no trade from Friday 09.00 NY until Sunday 03.00 NY.
func isNoTradeWindowNY(t time.Time) bool {
	// Special case. Sunday holiday during London session is allowed to trade
	// so we must explicitly opt out of the no trade window here.
	if t.Weekday() == time.Sunday && isLondonSession(t) {
		return t.Hour() < 3
	}

	// Full day block on holidays in all other cases
	if isHoliday(t) {
		return true
	}

	h := t.Hour()
	switch t.Weekday() {
	case time.Friday:
		return h >= 9
	case time.Saturday:
		return true
	case time.Sunday:
		return h < 3
	default:
		return false
	}
}

// detectSession uses exactly the same ordering as your original switch.
func detectSession(t time.Time) Session {
	if t.Weekday() == time.Sunday && isLondonSession(t) {
		return SessionLondon
	}

	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday || isHoliday(t) {
		return SessionWeekendHoliday
	}

	switch {
	case isDeadZone(t):
		return SessionDeadZone
	case isAsiaSession(t):
		return SessionAsia
	case isLondonSession(t):
		return SessionLondon
	case isUSSession(t):
		return SessionUS
	default:
		return SessionDefault
	}
}

// sizeMultiplierForSession just maps session to configured multiplier.
func sizeMultiplierForSession(s Session, cfg SessionSizeConfig) decimal.Decimal {
	switch s {
	case SessionWeekendHoliday:
		return cfg.WeekendHolidayMultiplier
	case SessionDeadZone:
		return cfg.DeadZoneMultiplier
	case SessionAsia:
		return cfg.AsiaMultiplier
	case SessionLondon:
		return cfg.LondonMultiplier
	case SessionUS:
		return cfg.USMultiplier
	default:
		return cfg.DefaultMultiplier
	}
}

func isDeadZone(t time.Time) bool {
	return t.Hour() >= 17 && t.Hour() < 20
}

func isAsiaSession(t time.Time) bool {
	return t.Hour() >= 20 || t.Hour() < 3
}

func isLondonSession(t time.Time) bool {
	return t.Hour() >= 3 && t.Hour() < 9
}

func isUSSession(t time.Time) bool {
	return t.Hour() >= 9 && t.Hour() <= 17
}

func isHoliday(t time.Time) bool {
	year := t.Year()

	// Calculate New Year's Day, adjusted for being on a Sunday
	newYearsDay := time.Date(year, time.January, NewYearDay, 0, 0, 0, 0, time.UTC)
	if newYearsDay.Weekday() == time.Sunday {
		newYearsDay = newYearsDay.AddDate(0, 0, OffsetDaysForNewYear)
	}

	// Martin Luther King Jr. Day and Presidents' Day calculation
	mlkDay := calculateSpecificMonday(year, time.January, ThirdMondayOffset)
	presidentsDay := calculateSpecificMonday(year, time.February, ThirdMondayOffset)

	// Memorial Day
	memorialDay := time.Date(year, time.May, 31, 0, 0, 0, 0, time.UTC)
	for memorialDay.Weekday() != time.Monday {
		memorialDay = memorialDay.AddDate(0, 0, -1)
	}

	// Independence Day
	independenceDay := time.Date(year, time.July, 4, 0, 0, 0, 0, time.UTC)
	if independenceDay.Weekday() == time.Sunday {
		independenceDay = independenceDay.AddDate(0, 0, OffsetDaysForNewYear)
	}

	// Labor Day
	laborDay := calculateSpecificMonday(year, time.September, 0)

	// Thanksgiving Day
	thanksgivingDay := calculateSpecificThursday(year, time.November, FourthThursdayOffset)

	// Christmas Day
	christmasDay := time.Date(year, time.December, 25, 0, 0, 0, 0, time.UTC)
	if christmasDay.Weekday() == time.Sunday {
		christmasDay = christmasDay.AddDate(0, 0, OffsetDaysForNewYear)
	}

	holidays := []time.Time{
		newYearsDay,
		mlkDay,
		presidentsDay,
		memorialDay,
		independenceDay,
		laborDay,
		thanksgivingDay,
		christmasDay,
	}
	return isDateAmong(t, holidays)
}

// calculateSpecificMonday calculates the specific Monday of a month (like the third Monday).
func calculateSpecificMonday(year int, month time.Month, mondayOffset int) time.Time {
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := int(time.Monday-firstOfMonth.Weekday()+DaysPerWeek) % DaysPerWeek
	return firstOfMonth.AddDate(0, 0, offset+mondayOffset*DaysPerWeek)
}

// calculateSpecificThursday calculates the specific Thursday of a month (like the fourth Thursday).
func calculateSpecificThursday(year int, month time.Month, thursdayOffset int) time.Time {
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := int(time.Thursday-firstOfMonth.Weekday()+DaysPerWeek) % DaysPerWeek
	return firstOfMonth.AddDate(0, 0, offset+thursdayOffset*DaysPerWeek)
}

// isDateAmong checks if the given date matches any date in the list.
func isDateAmong(t time.Time, dates []time.Time) bool {
	for _, d := range dates {
		if t.Format("2006-01-02") == d.Format("2006-01-02") {
			return true
		}
	}
	return false
}
