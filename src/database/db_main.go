package database

import (
	"fmt"
	"strategyexecutor/src/database/migrations"
	"strategyexecutor/src/model"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseConfig holds the basic configuration required to connect to a PostgreSQL database.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// MainDB is the primary read/write database connection used by the application.
var MainDB *gorm.DB

// InitMainDB initializes the main (read/write) database connection and runs migrations.
// This should be called once at application startup (e.g. in main()).
func InitMainDB() error {

	config := GetConfig()
	db, err := gorm.Open(postgres.Open(config.DatabaseURLMain),
		&gorm.Config{
			TranslateError: true,
			Logger:         logger.Default.LogMode(logger.LogLevel(config.GormLogLevel)),
		},
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get DB from GORM")
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	// Assign to the global variable only after a successful connection.
	MainDB = db

	logrus.Info("[database] MainDB connection established")

	// Prepare legacy user columns before AutoMigrate so we can safely
	// convert string-based user_id columns into numeric IDs without
	// failing casts.
	if err := migrations.PrepareLegacyUserColumns(MainDB); err != nil {
		return fmt.Errorf("failed to prepare legacy user columns: %w", err)
	}

	// Run AutoMigrate only on the main database.
	// Add here all models that belong to the write-side schema.
	if err := MainDB.AutoMigrate(
		&model.User{},
		&model.Order{},
		&model.OrderLog{},
		&model.OrderExecutionLog{},
		&model.Exchange{},
		&model.PhemexOrder{},
		&model.Exception{},
		&model.UserExchange{},
		&model.TradingViewNewsEvent{},
		&model.OHLCVCrypto1m{},
		&model.OHLCVCrypto1h{},
		&migrations.DataMigration{},
		//&model.Strategy{},
		//&model.StrategyAction{},
	); err != nil {
		return fmt.Errorf("failed to run migrations on MainDB: %w", err)
	}

	if err := migrations.Run(MainDB); err != nil {
		return fmt.Errorf("failed to run data migrations on MainDB: %w", err)
	}

	logrus.Info("[database] MainDB migrations completed")

	return nil
}
