package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vsC1Y2025V01/src/payloads"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"vsC1Y2025V01/src/auth"
	"vsC1Y2025V01/src/model"
	"vsC1Y2025V01/src/repository"
)

func CreateWebhookHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok || user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var payload payloads.CreateWebhookPayload
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			logger.WithError(err).Warn("invalid webhook payload")
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		payload.Name = strings.TrimSpace(payload.Name)
		payload.Type = strings.TrimSpace(payload.Type)
		if payload.Name == "" || payload.Type == "" {
			http.Error(w, "Name and type are required", http.StatusBadRequest)
			return
		}

		active := true
		if payload.Active != nil {
			active = *payload.Active
		}

		token, err := generateToken()
		if err != nil {
			logger.WithError(err).Error("failed to generate webhook token")
			http.Error(w, "Unable to create webhook", http.StatusInternalServerError)
			return
		}

		webhook := model.Webhook{
			UserID:      user.ID,
			Name:        payload.Name,
			Description: payload.Description,
			Style:       "",
			Type:        payload.Type,
			Tickers:     payload.Tickers,
			Token:       token,
			Active:      active,
			CreatedAt:   time.Time{},
			UpdatedAt:   time.Time{},
		}

		if err := repository.GetWebhookRepository().Create(&webhook); err != nil {
			logger.WithError(err).Error("failed to persist webhook")
			http.Error(w, "Unable to create webhook", http.StatusInternalServerError)
			return
		}

		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}

		response := map[string]interface{}{
			"webhook": webhook,
			"url":     fmt.Sprintf("%s://%s/trading/webhook/", scheme, r.Host),
			"token":   webhook.Token,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.WithError(err).Error("failed to encode webhook response")
		}
	}
}

func AlertHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		if strings.TrimSpace(token) == "" {
			http.Error(w, "Webhook token is required", http.StatusBadRequest)
			return
		}

		webhook, err := repository.GetWebhookRepository().FindByToken(token)
		if err != nil {
			if errors.Is(err, repository.ErrWebhookNotFound) {
				http.Error(w, "Webhook not found", http.StatusNotFound)
				return
			}
			logger.WithError(err).Error("failed to load webhook")
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		var payload payloads.AlertPayload
		decoder := json.NewDecoder(r.Body)
		//decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			logger.WithError(err).Warn("invalid alert payload")
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		var alertTime *time.Time
		if payload.AlertTime != "" {
			parsedTime, err := time.Parse(time.RFC3339, payload.AlertTime)
			if err != nil {
				logger.WithError(err).Warn("invalid alert Time")
				http.Error(w, "time must be RFC3339 formatted", http.StatusBadRequest)
				return
			}
			alertTime = &parsedTime
		}

		quantity, err := stringToFloat64(payload.Qty)
		if err != nil {
			logger.WithError(err).Warn("invalid alert quantity")
			http.Error(w, "quantity must be Float", http.StatusBadRequest)
			return
		}

		price, err := stringToFloat64(payload.Price)
		if err != nil {
			logger.WithError(err).Warn("invalid alert price")
			http.Error(w, "price must be Float", http.StatusBadRequest)
			return
		}

		marketPositionSize, err := stringToFloat64(payload.MarketPositionSize)
		if err != nil {
			logger.WithError(err).Warn("invalid alert Market Position Size")
			http.Error(w, "marketPositionSize must be Float", http.StatusBadRequest)
			return
		}

		prevMarketPositionSize, err := stringToFloat64(payload.PrevMarketPositionSize)
		if err != nil {
			logger.WithError(err).Warn("invalid alert Prev Market Position Size")
			http.Error(w, "prevMarketPositionSize must be Float", http.StatusBadRequest)
			return
		}

		alert := model.WebhookAlert{
			WebhookID:              webhook.ID,
			UserID:                 webhook.UserID,
			Ticker:                 payload.Ticker,
			Action:                 payload.Action,
			Sentiment:              payload.Sentiment,
			Quantity:               quantity,
			Price:                  price,
			Interval:               payload.Interval,
			MarketPosition:         payload.MarketPosition,
			PrevMarketPosition:     payload.PrevMarketPosition,
			MarketPositionSize:     marketPositionSize,
			PrevMarketPositionSize: prevMarketPositionSize,
			AlertTime:              alertTime,
			ReceivedAt:             time.Now(),
		}

		if err := repository.GetWebhookAlertRepository().Create(&alert); err != nil {
			logger.WithError(err).Error("failed to store webhook alert")
			http.Error(w, "Unable to store alert", http.StatusInternalServerError)
			return
		}

		//w.Header().Set("Content-Type", "application/json")
		//if err := json.NewEncoder(w).Encode(alert); err != nil {
		//	logger.WithError(err).Error("failed to encode alert response")
		//}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func ListWebhooksHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		offset, limit := ParseRangeParams(r)
		sortField, sortDir := ParseSortParams(r)
		filters := ParseFilterParams(r)

		var activeFilter *bool
		if activeStr, ok := filters["active"]; ok {
			active, err := strconv.ParseBool(activeStr)
			if err != nil {
				http.Error(w, "invalid active filter", http.StatusBadRequest)
				return
			}
			activeFilter = &active
			delete(filters, "active")
		}

		options := repository.WebhookListOptions{
			UserID:    user.ID,
			Active:    activeFilter,
			Filters:   filters,
			SortField: sortField,
			SortDir:   sortDir,
			Offset:    offset,
			Limit:     limit,
		}

		webhooks, total, err := repository.GetWebhookRepository().List(options)
		if err != nil {
			logger.WithError(err).Error("failed to fetch webhooks")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
		w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(webhooks); err != nil {
			logger.WithError(err).Error("failed to encode webhooks response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

func UpdateWebhookHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		idStr := chi.URLParam(r, "id")
		webhookID, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid webhook id", http.StatusBadRequest)
			return
		}

		webhook, err := repository.GetWebhookRepository().FindByIDForUser(uint(webhookID), user.ID)
		if err != nil {
			if errors.Is(err, repository.ErrWebhookNotFound) {
				http.Error(w, "Webhook not found", http.StatusNotFound)
				return
			}
			logger.WithError(err).Error("failed to load webhook")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var payload payloads.UpdateWebhookPayload
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			logger.WithError(err).Warn("invalid update webhook payload")
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if payload.Name != nil {
			trimmed := strings.TrimSpace(*payload.Name)
			if trimmed == "" {
				http.Error(w, "name cannot be empty", http.StatusBadRequest)
				return
			}
			webhook.Name = trimmed
		}

		if payload.Description != nil {
			webhook.Description = strings.TrimSpace(*payload.Description)
		}

		if payload.Type != nil {
			trimmed := strings.TrimSpace(*payload.Type)
			if trimmed == "" {
				http.Error(w, "type cannot be empty", http.StatusBadRequest)
				return
			}
			webhook.Type = trimmed
		}

		if payload.Tickers != nil {
			webhook.Tickers = *payload.Tickers
		}

		if payload.Active != nil {
			webhook.Active = *payload.Active
		}

		if err := repository.GetWebhookRepository().Save(webhook); err != nil {
			logger.WithError(err).Error("failed to update webhook")
			http.Error(w, "Unable to update webhook", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(webhook); err != nil {
			logger.WithError(err).Error("failed to encode updated webhook")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

func DeleteWebhookHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		idStr := chi.URLParam(r, "id")
		webhookID, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid webhook id", http.StatusBadRequest)
			return
		}

		webhook, err := repository.GetWebhookRepository().FindByIDForUser(uint(webhookID), user.ID)
		if err != nil {
			if errors.Is(err, repository.ErrWebhookNotFound) {
				http.Error(w, "Webhook not found", http.StatusNotFound)
				return
			}
			logger.WithError(err).Error("failed to load webhook for deletion")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err := repository.GetWebhookRepository().Delete(webhook); err != nil {
			logger.WithError(err).Error("failed to delete webhook")
			http.Error(w, "Unable to delete webhook", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func ListWebhookAlertsHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		offset, limit := ParseRangeParams(r)
		sortField, sortDir := ParseSortParams(r)
		filters := ParseFilterParams(r)

		var webhookID *uint
		if webhookIDStr, ok := filters["webhook_id"]; ok {
			id, err := strconv.ParseUint(webhookIDStr, 10, 64)
			if err != nil {
				http.Error(w, "invalid webhook_id filter", http.StatusBadRequest)
				return
			}
			typedID := uint(id)
			webhookID = &typedID
			delete(filters, "webhook_id")
		}

		options := repository.WebhookAlertListOptions{
			UserID:    user.ID,
			WebhookID: webhookID,
			Filters:   filters,
			SortField: sortField,
			SortDir:   sortDir,
			Offset:    offset,
			Limit:     limit,
		}

		alerts, total, err := repository.GetWebhookAlertRepository().List(options)
		if err != nil {
			logger.WithError(err).Error("failed to fetch webhook alerts")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
		w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(alerts); err != nil {
			logger.WithError(err).Error("failed to encode webhook alerts response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}
