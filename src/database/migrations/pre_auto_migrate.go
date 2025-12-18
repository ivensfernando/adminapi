package migrations

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// PrepareLegacyUserColumns normalizes schemas that previously stored user IDs as
// strings so that AutoMigrate can safely create bigint user_id columns without
// failing to cast legacy values.
func PrepareLegacyUserColumns(db *gorm.DB) error {
	tables := []string{"orders", "user_exchanges"}

	for _, table := range tables {
		columnType, exists, err := lookupColumnType(db, table, "user_id")
		if err != nil {
			return fmt.Errorf("inspect %s.user_id: %w", table, err)
		}

		legacyExists := false
		if _, existsLegacy, err := lookupColumnType(db, table, "legacy_user_id"); err == nil {
			legacyExists = existsLegacy
		}

		// If user_id is stored as a string column, preserve its data into the
		// legacy column and drop it so a fresh bigint column can be added.
		if exists && isStringy(columnType) {
			if !legacyExists {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN legacy_user_id varchar(255)", table)).Error; err != nil {
					return fmt.Errorf("add legacy_user_id to %s: %w", table, err)
				}
			}

			if err := db.Exec(fmt.Sprintf("UPDATE %s SET legacy_user_id = user_id WHERE user_id IS NOT NULL AND user_id <> ''", table)).Error; err != nil {
				return fmt.Errorf("backfill legacy_user_id on %s: %w", table, err)
			}

			if err := db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN user_id", table)).Error; err != nil {
				return fmt.Errorf("drop string user_id on %s: %w", table, err)
			}

			exists = false
		}

		// Ensure the numeric user_id column exists after cleanup.
		if !exists {
			if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN user_id bigint", table)).Error; err != nil {
				return fmt.Errorf("add bigint user_id to %s: %w", table, err)
			}
		}
	}

	return nil
}

func lookupColumnType(db *gorm.DB, table, column string) (dataType string, exists bool, err error) {
	row := db.Raw(
		`SELECT data_type FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
		table,
		column,
	).Row()

	if scanErr := row.Scan(&dataType); scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, scanErr
	}

	return dataType, true, nil
}

func isStringy(dataType string) bool {
	dataType = strings.ToLower(dataType)
	return strings.Contains(dataType, "char") || dataType == "text"
}
