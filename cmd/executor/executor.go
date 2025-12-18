package executor

import (
	"context"
	"os"
	"os/signal"
	"strategyexecutor/src/database"
	"strategyexecutor/src/executors"
	"syscall"

	"github.com/sirupsen/logrus"
)

func (t *Executor) Start() error {
	config := executors.GetConfig()
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	// Initialize main (read/write) database
	if err := database.InitMainDB(); err != nil {
		logrus.WithError(err).Fatal("Failed to connect to main database")
		return err
	}

	// Initialize read-only database
	if err := database.InitReadOnlyDB(); err != nil {
		logrus.WithError(err).Fatal("Failed to connect to read-only database")
		return err
	}

	targetExchange := config.TargetExchange
	logrus.WithField("targetExchange", targetExchange).Info("Starting strategy executor for exchange")

	if err := executors.StartLoop(ctx); err != nil {
		logrus.WithError(err).Error("Failed to start minute loop")
		return err
	}

	return nil
}
