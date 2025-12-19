// package migrations
package migrations

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DataMigration tracks executed data migrations (like Django).
// Table name is fixed to avoid collisions with other models.
type DataMigration struct {
	ID        string    `gorm:"primaryKey;size:200;column:id"`
	AppliedAt time.Time `gorm:"not null;column:applied_at"`
}

func (DataMigration) TableName() string { return "data_migrations" }

func ensureDataMigrationsTable(db *gorm.DB) error {
	return db.AutoMigrate()
}

// RunOnce runs fn only if migrationID was not executed before.
// It records the migration as executed only after fn succeeds.
func RunOnce(db *gorm.DB, migrationID string, fn func(*gorm.DB) error) error {
	if db == nil {
		return nil
	}
	if migrationID == "" {
		return fmt.Errorf("migration id is empty")
	}
	if fn == nil {
		return fmt.Errorf("migration %q has nil fn", migrationID)
	}

	if err := ensureDataMigrationsTable(db); err != nil {
		return fmt.Errorf("ensure data migrations table: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var m DataMigration
		err := tx.First(&m, "id = ?", migrationID).Error
		if err == nil {
			// already applied
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check migration %q: %w", migrationID, err)
		}

		// execute migration work
		if err := fn(tx); err != nil {
			return fmt.Errorf("run migration %q: %w", migrationID, err)
		}

		// record as applied
		rec := DataMigration{
			ID:        migrationID,
			AppliedAt: time.Now().UTC(),
		}
		if err := tx.Create(&rec).Error; err != nil {
			return fmt.Errorf("record migration %q: %w", migrationID, err)
		}

		return nil
	})
}

// Run executes all data migrations that go beyond schema auto-migrations.
// Append new migrations at the bottom with a stable unique id.
func Run(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	if err := RunOnce(db, "00001_migrate_legacy_users", migrateLegacyUsers); err != nil {
		return err
	}

	if err := RunOnce(db, "00002_backfill_user_exchange_session_size_defaults", backfillUserExchangeSessionSizeDefaults); err != nil {
		return err
	}

	if err := RunOnce(db, "00003_backfill_migrate_order_direction", migrateOrderDirection); err != nil {
		return err
	}

	if err := migrateOrderDirection(db); err != nil {
		return err
	}

	return nil
}
