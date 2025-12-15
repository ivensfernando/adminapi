package main

import (
	"adminapi/src/database"
	"adminapi/src/server"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

var (
	PORT     = os.Getenv("SERVER_PORT")
	APP_NAME = os.Getenv("APP_NAME")
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

	// Initialize main (read/write) database
	if err := database.InitMainDB(); err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}

	// Initialize read-only database
	if err := database.InitReadOnlyDB(); err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}

	server.StartServer(PORT)
}

func handlePanic() {
	if r := recover(); r != nil {
		logger.WithError(fmt.Errorf("%+v", r)).Error(fmt.Sprintf("Application %s panic", APP_NAME))
	}
	//nolint
	time.Sleep(time.Second * 5)
}
