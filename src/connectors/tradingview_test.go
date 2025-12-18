package connectors_test

import (
	"context"
	"strategyexecutor/src/connectors"
	"strategyexecutor/src/model"
	"strategyexecutor/src/repository"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// helper to create a new in memory gorm DB and migrate schema
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in memory db: %v", err)
	}

	if err := db.AutoMigrate(&model.TradingViewNewsEvent{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	return db
}

// helper to construct TVTime
func tvTime(t time.Time) model.TVTime {
	return model.TVTime{Time: t.UTC()}
}

// Test saving events from API representation and loading from DB,
// then using them with CanEnterTradeAt
func TestSaveLoadImportantEventsAndGate(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	repo := repository.NewTradingViewRepositoryWithDB(db)

	// base event time in UTC
	base := time.Date(2025, 12, 8, 16, 0, 0, 0, time.UTC)

	// sample events similar to the JSON you posted
	events := []model.Event{
		{
			ID:          "388335",
			Title:       "Consumer Inflation Expectations",
			Country:     "US",
			Indicator:   "Inflation Expectations",
			Ticker:      "ECONOMICS:USIE",
			Comment:     "some comment",
			Category:    "prce",
			Period:      "Nov",
			Source:      "Federal Reserve Bank of New York",
			SourceURL:   "https://www.newyorkfed.org",
			Actual:      floatPtr(3.2),
			Previous:    floatPtr(3.2),
			Forecast:    nil,
			ActualRaw:   floatPtr(3.2),
			PreviousRaw: floatPtr(3.2),
			ForecastRaw: nil,
			Currency:    "USD",
			Unit:        "%",
			Importance:  1, // important, must be loaded
			Date:        tvTime(base),
		},
		{
			ID:         "371430",
			Title:      "3 Month Bill Auction",
			Country:    "US",
			Indicator:  "3 Month Bill Yield",
			Category:   "bnd",
			Source:     "Federal Reserve",
			SourceURL:  "http://www.treasurydirect.gov",
			Actual:     floatPtr(3.65),
			Previous:   floatPtr(3.725),
			Currency:   "USD",
			Unit:       "%",
			Importance: -1, // not important, should be filtered out
			Date:       tvTime(base.Add(30 * time.Minute)),
		},
	}

	// save into DB
	if err := repo.SaveTradingViewNewsEvents(ctx, events); err != nil {
		t.Fatalf("SaveTradingViewNewsEvents failed: %v", err)
	}

	// load from DB within a window that covers both events
	from := base.Add(-1 * time.Hour)
	to := base.Add(2 * time.Hour)

	loaded, err := repo.LoadImportantEventsFromDB(ctx, from, to, []string{"US"})
	if err != nil {
		t.Fatalf("LoadImportantEventsFromDB failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 important event, got %d", len(loaded))
	}

	ev := loaded[0]
	if ev.ID != "388335" {
		t.Fatalf("expected event id 388335, got %s", ev.ID)
	}
	if ev.Importance != 1 {
		t.Fatalf("expected Importance 1, got %d", ev.Importance)
	}
	if ev.Date.Time.UTC() != base {
		t.Fatalf("expected Date %v, got %v", base, ev.Date.Time.UTC())
	}

	// now test the gate logic using the loaded event
	cfg := connectors.NewNewsWindowConfig(15*time.Minute, 15*time.Minute)

	// time inside block window
	nowBlocked := base.Add(5 * time.Minute)
	decisionBlocked := connectors.CanEnterTradeAt(nowBlocked, loaded, cfg)
	if decisionBlocked.Allowed {
		t.Fatalf("expected trade to be blocked by news window")
	}
	if decisionBlocked.BlockingEvent == nil {
		t.Fatalf("expected BlockingEvent to be set")
	}
	if decisionBlocked.BlockingEvent.ID != "388335" {
		t.Fatalf("expected BlockingEvent id 388335, got %s", decisionBlocked.BlockingEvent.ID)
	}

	expectedFrom := base.Add(-15 * time.Minute)
	expectedTo := base.Add(15 * time.Minute)
	if !decisionBlocked.BlockWindowFrom.Equal(expectedFrom) {
		t.Fatalf("unexpected BlockWindowFrom. got %v, want %v", decisionBlocked.BlockWindowFrom, expectedFrom)
	}
	if !decisionBlocked.BlockWindowTo.Equal(expectedTo) {
		t.Fatalf("unexpected BlockWindowTo. got %v, want %v", decisionBlocked.BlockWindowTo, expectedTo)
	}
	if !decisionBlocked.NextAllowedUTC.Equal(expectedTo) {
		t.Fatalf("unexpected NextAllowedUTC. got %v, want %v", decisionBlocked.NextAllowedUTC, expectedTo)
	}

	// time before block window
	nowBefore := base.Add(-20 * time.Minute)
	decisionBefore := connectors.CanEnterTradeAt(nowBefore, loaded, cfg)
	if !decisionBefore.Allowed {
		t.Fatalf("expected trade to be allowed before block window")
	}

	// time after block window
	nowAfter := base.Add(20 * time.Minute)
	decisionAfter := connectors.CanEnterTradeAt(nowAfter, loaded, cfg)
	if !decisionAfter.Allowed {
		t.Fatalf("expected trade to be allowed after block window")
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

func TestFetchImportantEvents_RealAPI(t *testing.T) {
	client := connectors.NewClientTV(nil)

	ctx := context.Background()

	// A reasonable window: yesterday → tomorrow
	from := time.Now().Add(-24 * time.Hour).UTC()
	to := time.Now().Add(24 * time.Hour).UTC()

	evs, err := client.FetchImportantEvents(ctx, from, to, []string{"US"})
	if err != nil {
		t.Fatalf("FetchImportantEvents failed: %v", err)
	}

	t.Logf("Fetched %d events", len(evs))

	// Sanity checks
	for _, ev := range evs {
		if ev.ID == "" {
			t.Fatalf("event id empty: %+v", ev)
		}
		if ev.Importance != 1 {
			t.Fatalf("found non-important event in importance==1 filter: %+v", ev)
		}
		if ev.Date.Time.IsZero() {
			t.Fatalf("event has zero date: %+v", ev)
		}

		t.Logf("Event: %s at %s", ev.Title, ev.Date.Time)
	}
}
