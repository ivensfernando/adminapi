package tv_news

import (
	"context"
	"strategyexecutor/src/connectors"
	"strategyexecutor/src/database"
	"strategyexecutor/src/repository"
	"time"

	"github.com/sirupsen/logrus"
)

func (t *TVNews) Start() error {
	if err := database.InitMainDB(); err != nil {
		logrus.WithError(err).Fatal("Failed to connect to database")
	}

	client := connectors.NewClientTV(nil)

	ctx := context.Background()

	// A reasonable window: yesterday → tomorrow
	from := time.Now().Add(-24 * time.Hour).UTC()
	to := time.Now().Add(7 * 24 * time.Hour).UTC()

	evs, err := client.FetchImportantEvents(ctx, from, to, []string{"US"})
	if err != nil {
		return err
	}

	logrus.Infof("Fetched %d events", len(evs))

	repo := repository.NewTradingViewRepository()

	// save into DB
	if err := repo.SaveTradingViewNewsEvents(ctx, evs); err != nil {
		return err
	}

	return nil
}
