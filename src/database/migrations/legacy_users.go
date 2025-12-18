package migrations

import (
	"errors"
	"fmt"
	"time"

	"strategyexecutor/src/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	legacyDefaultPassword = "123"
)

// migrateLegacyUsers backfills legacy string-based user identifiers into the new users table
// and rewires dependent records to use the new numeric keys. A default password is generated
// for each migrated user to keep compatibility with authentication flows.
func migrateLegacyUsers(db *gorm.DB) error {
	legacyIDs, err := collectLegacyUserIDs(db)
	if err != nil {
		return fmt.Errorf("collect legacy user ids: %w", err)
	}

	for legacyID := range legacyIDs {
		userID, err := ensureUserForLegacyID(db, legacyID)
		if err != nil {
			return fmt.Errorf("ensure user for legacy_id %s: %w", legacyID, err)
		}

		if err := migrateOrderLegacyIDs(db, legacyID, userID); err != nil {
			return fmt.Errorf("migrate orders for legacy_id %s: %w", legacyID, err)
		}

		if err := migrateUserExchangeLegacyIDs(db, legacyID, userID); err != nil {
			return fmt.Errorf("migrate user exchanges for legacy_id %s: %w", legacyID, err)
		}
	}

	return nil
}

func collectLegacyUserIDs(db *gorm.DB) (map[string]struct{}, error) {
	legacyIDs := make(map[string]struct{})

	var orderIDs []string
	if err := db.Model(&model.Order{}).
		Distinct().
		Where("legacy_user_id IS NOT NULL AND legacy_user_id <> ''").
		Pluck("legacy_user_id", &orderIDs).Error; err != nil {
		return nil, err
	}

	for _, id := range orderIDs {
		legacyIDs[id] = struct{}{}
	}

	var userExchangeIDs []string
	if err := db.Model(&model.UserExchange{}).
		Distinct().
		Where("legacy_user_id IS NOT NULL AND legacy_user_id <> ''").
		Pluck("legacy_user_id", &userExchangeIDs).Error; err != nil {
		return nil, err
	}

	for _, id := range userExchangeIDs {
		legacyIDs[id] = struct{}{}
	}

	return legacyIDs, nil
}

func ensureUserForLegacyID(db *gorm.DB, legacyID string) (uint, error) {
	var user model.User
	if err := db.Where("user_name = ?", legacyID).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(legacyDefaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return 0, fmt.Errorf("hash default password: %w", err)
		}

		user = model.User{
			Username:  legacyID,
			Password:  string(hashedPassword),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			return 0, err
		}
	}

	return user.ID, nil
}

func migrateOrderLegacyIDs(db *gorm.DB, legacyID string, userID uint) error {
	return db.Model(&model.Order{}).
		Where("legacy_user_id = ? AND (user_id IS NULL OR user_id = 0)", legacyID).
		Update("user_id", userID).Error
}

func migrateUserExchangeLegacyIDs(db *gorm.DB, legacyID string, userID uint) error {
	return db.Model(&model.UserExchange{}).
		Where("legacy_user_id = ? AND (user_id IS NULL OR user_id = 0)", legacyID).
		Update("user_id", userID).Error
}
