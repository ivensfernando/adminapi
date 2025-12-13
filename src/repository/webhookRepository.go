package repository

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"vsC1Y2025V01/src/db"
	"vsC1Y2025V01/src/model"

	"gorm.io/gorm"
)

var ErrWebhookNotFound = errors.New("webhook not found")

type WebhookListOptions struct {
	UserID    uint
	Active    *bool
	Filters   map[string]string
	SortField string
	SortDir   string
	Offset    int
	Limit     int
}

type WebhookRepository interface {
	Create(webhook *model.Webhook) error
	FindByToken(token string) (*model.Webhook, error)
	FindByIDForUser(id, userID uint) (*model.Webhook, error)
	List(options WebhookListOptions) ([]model.Webhook, int64, error)
	Save(webhook *model.Webhook) error
	Delete(webhook *model.Webhook) error
}

type WebhookAlertListOptions struct {
	UserID    uint
	WebhookID *uint
	Filters   map[string]string
	SortField string
	SortDir   string
	Offset    int
	Limit     int
}

type WebhookAlertRepository interface {
	Create(alert *model.WebhookAlert) error
	List(options WebhookAlertListOptions) ([]model.WebhookAlert, int64, error)
}

var (
	webhookRepo   WebhookRepository = &gormWebhookRepository{}
	webhookRepoMu sync.RWMutex
	alertRepo     WebhookAlertRepository = &gormWebhookAlertRepository{}
	alertRepoMu   sync.RWMutex
)

//func SetWebhookRepository(repo WebhookRepository) {
//	webhookRepoMu.Lock()
//	defer webhookRepoMu.Unlock()
//
//	if repo == nil {
//		webhookRepo = &gormWebhookRepository{}
//		return
//	}
//
//	webhookRepo = repo
//}

func GetWebhookRepository() WebhookRepository {
	webhookRepoMu.RLock()
	repo := webhookRepo
	webhookRepoMu.RUnlock()

	if repo != nil {
		return repo
	}

	webhookRepoMu.Lock()
	defer webhookRepoMu.Unlock()

	if webhookRepo == nil {
		webhookRepo = &gormWebhookRepository{}
	}

	return webhookRepo
}

//func SetWebhookAlertRepository(repo WebhookAlertRepository) {
//	alertRepoMu.Lock()
//	defer alertRepoMu.Unlock()
//
//	if repo == nil {
//		alertRepo = &gormWebhookAlertRepository{}
//		return
//	}
//
//	alertRepo = repo
//}

func GetWebhookAlertRepository() WebhookAlertRepository {
	alertRepoMu.RLock()
	repo := alertRepo
	alertRepoMu.RUnlock()

	if repo != nil {
		return repo
	}

	alertRepoMu.Lock()
	defer alertRepoMu.Unlock()

	if alertRepo == nil {
		alertRepo = &gormWebhookAlertRepository{}
	}

	return alertRepo
}

type gormWebhookRepository struct{}

func (r *gormWebhookRepository) Create(webhook *model.Webhook) error {
	if db.DB == nil {
		return errors.New("database connection is not initialized")
	}

	return db.DB.Create(webhook).Error
}

func (r *gormWebhookRepository) FindByToken(token string) (*model.Webhook, error) {
	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	var webhook model.Webhook
	if err := db.DB.Where("token = ?", token).First(&webhook).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWebhookNotFound
		}

		return nil, err
	}

	return &webhook, nil
}

func (r *gormWebhookRepository) FindByIDForUser(id, userID uint) (*model.Webhook, error) {
	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	var webhook model.Webhook
	if err := db.DB.Where("id = ? AND user_id = ?", id, userID).First(&webhook).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWebhookNotFound
		}

		return nil, err
	}

	return &webhook, nil
}

func (r *gormWebhookRepository) List(options WebhookListOptions) ([]model.Webhook, int64, error) {
	if db.DB == nil {
		return nil, 0, errors.New("database connection is not initialized")
	}

	query := db.DB.Model(&model.Webhook{}).Where("user_id = ?", options.UserID)

	if options.Active != nil {
		query = query.Where("active = ?", *options.Active)
	}

	for k, v := range options.Filters {
		query = query.Where(fmt.Sprintf("%s = ?", k), v)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var webhooks []model.Webhook
	if err := query.Order(fmt.Sprintf("%s %s", options.SortField, options.SortDir)).
		Offset(options.Offset).Limit(options.Limit).
		Find(&webhooks).Error; err != nil {
		return nil, 0, err
	}

	return webhooks, total, nil
}

func (r *gormWebhookRepository) Save(webhook *model.Webhook) error {
	if db.DB == nil {
		return errors.New("database connection is not initialized")
	}

	if webhook != nil {
		webhook.UpdatedAt = time.Now()
	}

	return db.DB.Save(webhook).Error
}

func (r *gormWebhookRepository) Delete(webhook *model.Webhook) error {
	if db.DB == nil {
		return errors.New("database connection is not initialized")
	}

	return db.DB.Delete(webhook).Error
}

type gormWebhookAlertRepository struct{}

func (r *gormWebhookAlertRepository) Create(alert *model.WebhookAlert) error {
	if db.DB == nil {
		return errors.New("database connection is not initialized")
	}

	return db.DB.Create(alert).Error
}

func (r *gormWebhookAlertRepository) List(options WebhookAlertListOptions) ([]model.WebhookAlert, int64, error) {
	if db.DB == nil {
		return nil, 0, errors.New("database connection is not initialized")
	}

	query := db.DB.Model(&model.WebhookAlert{}).Where("user_id = ?", options.UserID)

	if options.WebhookID != nil {
		query = query.Where("webhook_id = ?", *options.WebhookID)
	}

	for k, v := range options.Filters {
		query = query.Where(fmt.Sprintf("%s = ?", k), v)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var alerts []model.WebhookAlert
	if err := query.Order(fmt.Sprintf("%s %s", options.SortField, options.SortDir)).
		Offset(options.Offset).Limit(options.Limit).
		Find(&alerts).Error; err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}
