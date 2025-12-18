package repository

import (
	"context"
	"strategyexecutor/src/database"
	"strategyexecutor/src/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TradingViewRepository struct {
	db *gorm.DB
}

// NewTradingViewRepository creates a new repository instance.
func NewTradingViewRepository() *TradingViewRepository {
	//return &ExceptionRepository{db: db}
	return &TradingViewRepository{
		db: database.MainDB,
	}
}

func NewTradingViewRepositoryWithDB(db *gorm.DB) *TradingViewRepository {
	//return &ExceptionRepository{db: db}
	return &TradingViewRepository{
		db: db,
	}
}

// SaveTradingViewNewsEvents upserts all events based on tv_event_id
func (r *TradingViewRepository) SaveTradingViewNewsEvents(ctx context.Context, events []model.Event) error {
	for _, ev := range events {
		m := model.NewTradingViewNewsEventFromEvent(ev)

		// upsert on tv_event_id
		if err := r.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "tv_event_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"title",
					"country",
					"indicator",
					"ticker",
					"comment",
					"category",
					"period",
					"reference_date",
					"source",
					"source_url",
					"actual",
					"previous",
					"forecast",
					"actual_raw",
					"previous_raw",
					"forecast_raw",
					"currency",
					"unit",
					"importance",
					"event_date",
					"updated_at",
				}),
			}).
			Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

// LoadImportantEventsFromDB replicates the FetchImportantEvents logic but from Postgres
func (r *TradingViewRepository) LoadImportantEventsFromDB(
	ctx context.Context,
	fromUTC time.Time,
	toUTC time.Time,
	countries []string,
) ([]model.Event, error) {
	var rows []model.TradingViewNewsEvent

	q := r.db.WithContext(ctx).
		Where("event_date >= ? AND event_date <= ?", fromUTC.UTC(), toUTC.UTC()).
		Where("importance = ?", 1)

	if len(countries) > 0 {
		q = q.Where("country IN ?", countries)
	}

	if err := q.Order("event_date ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	evs := make([]model.Event, 0, len(rows))
	for _, m := range rows {
		evs = append(evs, m.ToEvent())
	}
	return evs, nil
}
