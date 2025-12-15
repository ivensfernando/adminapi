package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"adminapi/src/auth"
	"adminapi/src/model"
	"adminapi/src/repository"

	logger "github.com/sirupsen/logrus"
)

type orderSearcher interface {
	Search(ctx context.Context, options repository.OrderSearchOptions) ([]model.Order, error)
}

// SearchOrdersHandler returns a handler that lists orders for the authenticated user.
// Supports pagination and filters (exchangeId, symbol, createdFrom, createdTo).
func SearchOrdersHandler(repo orderSearcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok || user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var exchangeID *uint
		if exchangeParam := r.URL.Query().Get("exchangeId"); exchangeParam != "" {
			id, err := strconv.ParseUint(exchangeParam, 10, 64)
			if err != nil {
				http.Error(w, "invalid exchangeId", http.StatusBadRequest)
				return
			}
			exchange := uint(id)
			exchangeID = &exchange
		}

		var symbol *string
		if symbolParam := r.URL.Query().Get("symbol"); symbolParam != "" {
			symbol = &symbolParam
		}

		var createdFrom, createdTo *time.Time
		if createdFromParam := r.URL.Query().Get("createdFrom"); createdFromParam != "" {
			parsed, err := time.Parse(time.RFC3339, createdFromParam)
			if err != nil {
				http.Error(w, "invalid createdFrom", http.StatusBadRequest)
				return
			}
			createdFrom = &parsed
		}

		if createdToParam := r.URL.Query().Get("createdTo"); createdToParam != "" {
			parsed, err := time.Parse(time.RFC3339, createdToParam)
			if err != nil {
				http.Error(w, "invalid createdTo", http.StatusBadRequest)
				return
			}
			createdTo = &parsed
		}

		page := 1
		if pageParam := r.URL.Query().Get("page"); pageParam != "" {
			parsedPage, err := strconv.Atoi(pageParam)
			if err != nil || parsedPage <= 0 {
				http.Error(w, "invalid page", http.StatusBadRequest)
				return
			}
			page = parsedPage
		}

		pageSize := 20
		if sizeParam := r.URL.Query().Get("pageSize"); sizeParam != "" {
			parsedSize, err := strconv.Atoi(sizeParam)
			if err != nil || parsedSize <= 0 {
				http.Error(w, "invalid pageSize", http.StatusBadRequest)
				return
			}
			pageSize = parsedSize
		}

		offset := (page - 1) * pageSize

		orders, err := repo.Search(r.Context(), repository.OrderSearchOptions{
			UserID:        user.ID,
			ExchangeID:    exchangeID,
			Symbol:        symbol,
			CreatedAfter:  createdFrom,
			CreatedBefore: createdTo,
			Limit:         pageSize,
			Offset:        offset,
		})
		if err != nil {
			logger.WithError(err).Error("failed to search orders")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(orders); err != nil {
			logger.WithError(err).Error("failed to encode order search response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// DefaultSearchOrdersHandler wires the handler to the production repository implementation.
func DefaultSearchOrdersHandler() http.HandlerFunc {
	return SearchOrdersHandler(repository.NewOrderRepository())
}
