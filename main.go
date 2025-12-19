package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strategyexecutor/src/database"
	"strategyexecutor/src/executors"
	"strings"
	"syscall"
	"time"

	logger "github.com/sirupsen/logrus"
)

func SetupLogger() {
	levelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))

	level, err := logger.ParseLevel(levelStr)
	if err != nil {
		level = logger.DebugLevel // fallback seguro
	}

	logger.SetLevel(level)
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	SetupLogger()
	//db.InitDB(log) // ✅ MUST be here before any DB access
	defer handlePanic()

	//config := server.GetConfig()

	// Initialize main (read/write) database
	if err := database.InitMainDB(); err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}

	// Initialize read-only database
	if err := database.InitReadOnlyDB(); err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}

	//server.StartServer(config.Port)
	//config := executors.GetConfig()
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	targetExchange := "phemex"
	logger.WithField("targetExchange", targetExchange).Info("Starting strategy executor for exchange")

	if err := executors.StartLoop(ctx); err != nil {
		logger.WithError(err).Error("Failed to start minute loop")
		return
	}
}

func handlePanic() {
	if r := recover(); r != nil {
		logger.WithError(fmt.Errorf("%+v", r)).Error(fmt.Sprintf("Application panic"))
	}
	//nolint
	time.Sleep(time.Second * 5)
}
