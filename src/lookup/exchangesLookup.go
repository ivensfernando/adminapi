package lookup

import (
	"adminapi/src/database"
	"adminapi/src/model"
	"encoding/json"
	"fmt"
	"net/http"

	logger "github.com/sirupsen/logrus"
)

var (
	fetchExchanges = func() ([]model.Exchange, int64, error) {
		var exchanges []model.Exchange
		var total int64

		if err := db.DB.Model(&model.Exchange{}).Count(&total).Error; err != nil {
			return nil, 0, err
		}

		if err := db.DB.Find(&exchanges).Error; err != nil {
			return nil, 0, err
		}

		return exchanges, total, nil
	}

	fetchPairs = func() ([]model.PairsCoins, int64, error) {
		var pairs []model.PairsCoins
		var total int64

		if err := db.DB.Model(&model.PairsCoins{}).Count(&total).Error; err != nil {
			return nil, 0, err
		}

		if err := db.DB.Find(&pairs).Error; err != nil {
			return nil, 0, err
		}

		return pairs, total, nil
	}
)

// GET /exchanges
func ListExchanges() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchanges, total, err := fetchExchanges()
		if err != nil {
			logger.WithError(err).Error("Failed to count exchanges")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
		w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(exchanges); err != nil {
			logger.WithError(err).Error("Failed to encode exchanges to JSON")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return

		}
	}
}

// GET /pairs
func ListPairs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pairs, total, err := fetchPairs()
		if err != nil {
			logger.WithError(err).Error("Failed to count exchanges")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
		w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(pairs); err != nil {
			logger.WithError(err).Error("Failed to encode pairs to JSON")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}
