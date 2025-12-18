package migrations

import "gorm.io/gorm"

// Run executes all data migrations that go beyond schema auto-migrations.
// New migrations should be appended inside this function.
func Run(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	if err := migrateLegacyUsers(db); err != nil {
		return err
	}

	return nil
}
