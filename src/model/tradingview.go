package model

import (
	"fmt"
	"strconv"
	"time"
)

type EventsResponse struct {
	Status string  `json:"status"`
	Result []Event `json:"result"`
}

type Event struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Country       string   `json:"country"`
	Indicator     string   `json:"indicator"`
	Ticker        string   `json:"ticker"`
	Comment       string   `json:"comment"`
	Category      string   `json:"category"`
	Period        string   `json:"period"`
	ReferenceDate *TVTime  `json:"referenceDate"`
	Source        string   `json:"source"`
	SourceURL     string   `json:"source_url"`
	Actual        *float64 `json:"actual"`
	Previous      *float64 `json:"previous"`
	Forecast      *float64 `json:"forecast"`
	ActualRaw     *float64 `json:"actualRaw"`
	PreviousRaw   *float64 `json:"previousRaw"`
	ForecastRaw   *float64 `json:"forecastRaw"`
	Currency      string   `json:"currency"`
	Unit          string   `json:"unit"`
	Importance    int      `json:"importance"`
	Date          TVTime   `json:"date"`
}

// TVTime handles TradingView timestamps like:
// - "2025-12-08T16:00:00.000Z"
// - "2025-11-30T00:00:00Z"
type TVTime struct {
	time.Time
}

func (t *TVTime) UnmarshalJSON(b []byte) error {
	// null
	if string(b) == "null" {
		t.Time = time.Time{}
		return nil
	}

	s, err := strconv.Unquote(string(b))
	if err != nil {
		return fmt.Errorf("TVTime: invalid json string: %w", err)
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}

	// Try the common layouts TradingView returns.
	layouts := []string{
		"2006-01-02T15:04:05.000Z",
		time.RFC3339, // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05Z",
	}

	var lastErr error
	for _, layout := range layouts {
		tt, e := time.Parse(layout, s)
		if e == nil {
			t.Time = tt
			return nil
		}
		lastErr = e
	}
	return fmt.Errorf("TVTime: parse %q: %w", s, lastErr)
}

// TradingViewNewsEvent GORM VO for DB storage
type TradingViewNewsEvent struct {
	ID uint `gorm:"primaryKey"` // internal DB id

	TVEventID string `gorm:"column:tv_event_id;uniqueIndex;not null"` // TradingView "id"

	Title     string `gorm:"column:title"`
	Country   string `gorm:"column:country;index"`
	Indicator string `gorm:"column:indicator"`
	Ticker    string `gorm:"column:ticker"`
	Comment   string `gorm:"column:comment"`
	Category  string `gorm:"column:category"`
	Period    string `gorm:"column:period"`

	ReferenceDate *time.Time `gorm:"column:reference_date"`

	Source    string `gorm:"column:source"`
	SourceURL string `gorm:"column:source_url"`

	Actual      *float64 `gorm:"column:actual"`
	Previous    *float64 `gorm:"column:previous"`
	Forecast    *float64 `gorm:"column:forecast"`
	ActualRaw   *float64 `gorm:"column:actual_raw"`
	PreviousRaw *float64 `gorm:"column:previous_raw"`
	ForecastRaw *float64 `gorm:"column:forecast_raw"`

	Currency   string `gorm:"column:currency"`
	Unit       string `gorm:"column:unit"`
	Importance int    `gorm:"column:importance;index"`

	EventDate time.Time `gorm:"column:event_date;index"` // your main time for blocking

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (TradingViewNewsEvent) TableName() string {
	return "trading_view_news_event"
}

// NewTradingViewNewsEventFromEvent Converter from API Event into DB VO
func NewTradingViewNewsEventFromEvent(ev Event) TradingViewNewsEvent {
	var refDate *time.Time
	if ev.ReferenceDate != nil && !ev.ReferenceDate.Time.IsZero() {
		t := ev.ReferenceDate.Time.UTC()
		refDate = &t
	}

	return TradingViewNewsEvent{
		TVEventID:     ev.ID,
		Title:         ev.Title,
		Country:       ev.Country,
		Indicator:     ev.Indicator,
		Ticker:        ev.Ticker,
		Comment:       ev.Comment,
		Category:      ev.Category,
		Period:        ev.Period,
		ReferenceDate: refDate,
		Source:        ev.Source,
		SourceURL:     ev.SourceURL,
		Actual:        ev.Actual,
		Previous:      ev.Previous,
		Forecast:      ev.Forecast,
		ActualRaw:     ev.ActualRaw,
		PreviousRaw:   ev.PreviousRaw,
		ForecastRaw:   ev.ForecastRaw,
		Currency:      ev.Currency,
		Unit:          ev.Unit,
		Importance:    ev.Importance,
		EventDate:     ev.Date.Time.UTC(),
	}
}

// TradingViewNewsEvent Optional converter back from VO to API Event. Useful for CanEnterTradeAt
func (m TradingViewNewsEvent) ToEvent() Event {
	var ref *TVTime
	if m.ReferenceDate != nil && !m.ReferenceDate.IsZero() {
		ref = &TVTime{Time: m.ReferenceDate.UTC()}
	}

	return Event{
		ID:            m.TVEventID,
		Title:         m.Title,
		Country:       m.Country,
		Indicator:     m.Indicator,
		Ticker:        m.Ticker,
		Comment:       m.Comment,
		Category:      m.Category,
		Period:        m.Period,
		ReferenceDate: ref,
		Source:        m.Source,
		SourceURL:     m.SourceURL,
		Actual:        m.Actual,
		Previous:      m.Previous,
		Forecast:      m.Forecast,
		ActualRaw:     m.ActualRaw,
		PreviousRaw:   m.PreviousRaw,
		ForecastRaw:   m.ForecastRaw,
		Currency:      m.Currency,
		Unit:          m.Unit,
		Importance:    m.Importance,
		Date:          TVTime{Time: m.EventDate.UTC()},
	}
}
