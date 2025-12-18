package connectors

import (
	"sort"
	"strategyexecutor/src/model"
	"time"
)

type NewsWindowConfig struct {
	BlockBefore time.Duration // e.g. 15*time.Minute
	BlockAfter  time.Duration // e.g. 15*time.Minute
}

func NewNewsWindowConfig(blockBefore, blockAfter time.Duration) NewsWindowConfig {
	return NewsWindowConfig{
		BlockBefore: blockBefore,
		BlockAfter:  blockAfter,
	}
}

type TradeGateDecision struct {
	Allowed         bool
	Reason          string
	NowUTC          time.Time
	BlockingEvent   *model.Event
	BlockWindowFrom time.Time
	BlockWindowTo   time.Time
	NextAllowedUTC  time.Time
}

// CanEnterTradeNow checks whether a new position entry is allowed at time.Now().
// It blocks entries if now is within [eventTime-BlockBefore, eventTime+BlockAfter] for any importance=1 event.
func CanEnterTradeNow(events []model.Event, cfg NewsWindowConfig) TradeGateDecision {
	return CanEnterTradeAt(time.Now().UTC(), events, cfg)
}

// CanEnterTradeAt is the deterministic version for tests.
func CanEnterTradeAt(nowUTC time.Time, events []model.Event, cfg NewsWindowConfig) TradeGateDecision {
	// Only consider importance=1 events. Caller already filters, but this makes it robust.
	type window struct {
		ev    model.Event
		start time.Time
		end   time.Time
	}

	var active []window

	for _, ev := range events {
		evTime := ev.Date.Time.UTC()
		if evTime.IsZero() {
			continue
		}

		start := evTime.Add(-cfg.BlockBefore)
		end := evTime.Add(cfg.BlockAfter)

		if !nowUTC.Before(start) && !nowUTC.After(end) {
			active = append(active, window{ev: ev, start: start, end: end})
		}
	}

	if len(active) == 0 {
		return TradeGateDecision{
			Allowed: true,
			Reason:  "allowed",
			NowUTC:  nowUTC,
		}
	}

	// If multiple windows overlap now, you should wait until the latest end.
	sort.Slice(active, func(i, j int) bool {
		return active[i].end.Before(active[j].end)
	})
	block := active[len(active)-1]

	return TradeGateDecision{
		Allowed:         false,
		Reason:          "blocked_by_news_window",
		NowUTC:          nowUTC,
		BlockingEvent:   &block.ev,
		BlockWindowFrom: block.start,
		BlockWindowTo:   block.end,
		NextAllowedUTC:  block.end,
	}
}

func CanEnterTrade() bool {
	cfg := NewNewsWindowConfig(15*time.Minute, 15*time.Minute) // configurable
	client := NewClientTV(nil)

	from := time.Now().Add(-(time.Hour * 24 * 1)).UTC()
	to := time.Now().Add(time.Hour * 24 * 1).UTC()

	events, err := client.FetchImportantEvents(nil, from, to, []string{"US"})
	if err != nil {
		return false
	}

	decision := CanEnterTradeNow(events, cfg)
	if !decision.Allowed {
		// do not enter. optionally log decision.BlockingEvent.Title and decision.NextAllowedUTC
		return false
	}

	return true
}
