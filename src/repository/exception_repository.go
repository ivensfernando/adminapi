package repository

import (
	"adminapi/src/database"
	"context"

	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"adminapi/src/model"
)

// ExceptionRepository handles persistence of system exceptions.
type ExceptionRepository struct {
	db *gorm.DB
}

// NewExceptionRepository creates a new repository instance.
func NewExceptionRepository() *ExceptionRepository {
	//return &ExceptionRepository{database: database}
	return &ExceptionRepository{
		db: database.MainDB,
	}
}

// Create persists a new exception in the database.
func (r *ExceptionRepository) Create(
	ctx context.Context,
	exc *model.Exception,
) error {

	logger.WithFields(map[string]interface{}{
		"service": exc.Service,
		"module":  exc.Module,
		"method":  exc.Method,
		"level":   exc.Level,
	}).Error("Persisting system exception")

	return r.db.WithContext(ctx).Create(exc).Error
}
