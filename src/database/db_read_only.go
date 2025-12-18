package database

import (
	"fmt"
	"strategyexecutor/src/externalmodel"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	//"strategyexecutor/src/externalmodel"

	"github.com/sirupsen/logrus"
)

// ReadOnlyDB is the read-only database connection used to poll external trading signals.
// The database user for this connection should have SELECT-only permissions.
var ReadOnlyDB *gorm.DB

// InitReadOnlyDB initializes the read-only database connection.
// It does not run any migrations and should only be used for reading data.
func InitReadOnlyDB() error {
	config := GetConfig()
	db, err := gorm.Open(postgres.Open(config.DatabaseURLReadOnly),
		&gorm.Config{
			PrepareStmt:    true,
			TranslateError: true,
			Logger:         logger.Default.LogMode(logger.LogLevel(config.GormLogLevel)),
		},
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to database")
	}

	//_ = externalmodel.TradingSignal{}

	// ✅ Get low-level sql.DB from GORM
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from ReadOnlyDB: %w", err)
	}

	// ✅ REAL ping to the database
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping ReadOnlyDB: %w", err)
	}

	// ✅ Log current database and schema
	var dbName, schema string
	if err := db.
		Raw("SELECT current_database(), current_schema()").
		Row().
		Scan(&dbName, &schema); err != nil {
		return fmt.Errorf("failed to query current db/schema on ReadOnlyDB: %w", err)
	}

	logrus.WithFields(map[string]interface{}{"dbName": dbName, "schema": schema}).Info("[ReadOnlyDB] connected to database=%s schema=%s")

	//// ✅ Test if the table is really reachable
	var count int64
	if err1 := db.
		Model(&externalmodel.TradingSignal{}).
		Count(&count).Error; err1 != nil {

		return fmt.Errorf("failed to access trade_tradingsignal: %w", err1)
	}

	logrus.WithFields(map[string]interface{}{"count": count}).Info("[ReadOnlyDB] trade_tradingsignal reachable, total rows: %d")

	ReadOnlyDB = db

	return nil
}
