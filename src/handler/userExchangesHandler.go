package handler

import (
	"adminapi/src/repository"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	//"adminapi/internal/connectors"
	"adminapi/src/auth"
	"adminapi/src/model"
	"adminapi/src/security"

	"github.com/go-chi/chi/v5"
	logger "github.com/sirupsen/logrus"
)

type mexcConnector interface {
	TestConnection() error
}

//var mexcConnectorFactory = func(apiKey, apiSecret string) mexcConnector {
//	//return connectors.NewMexcConnector(apiKey, apiSecret)
//}
//
//var newMexcConnector = func(apiKey, apiSecret string) mexcConnector {
//	//return connectors.NewMexcConnector(apiKey, apiSecret)
//}

type testConnectionPayload struct {
	APIKey        string `json:"apiKey"`
	APISecret     string `json:"apiSecret"`
	APIPassphrase string `json:"apiPassphrase"`
}

type testConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func UpsertUserExchangeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok || user == nil {
			logger.Warn("user not found in context while upserting user exchange")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var payload model.UpsertUserExchangePayload
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			logger.WithError(err).Warn("invalid user exchange payload")
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if payload.ExchangeID == 0 {
			http.Error(w, "exchangeId is required", http.StatusBadRequest)
			return
		}

		payload.APIKey = strings.TrimSpace(payload.APIKey)
		payload.APISecret = strings.TrimSpace(payload.APISecret)
		payload.APIPassphrase = strings.TrimSpace(payload.APIPassphrase)

		exchange, err := repository.GetUserExchangeStore().GetExchangeByID(payload.ExchangeID)
		if err != nil {
			if errors.Is(err, repository.ErrExchangeNotFound) {
				http.Error(w, "exchange not found", http.StatusBadRequest)
				return
			}

			logger.WithError(err).Error("failed to load exchange for user exchange upsert")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		userExchange, err := repository.GetUserExchangeStore().FindUserExchange(user.ID, payload.ExchangeID)
		if err != nil {
			if errors.Is(err, repository.ErrUserExchangeNotFound) {
				userExchange = &model.UserExchange{
					//UserID:     user.ID,
					ExchangeID: payload.ExchangeID,
				}
			} else {
				logger.WithError(err).Error("failed to fetch user exchange")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		if payload.APIKey != "" {
			cipherText, err := security.EncryptString(payload.APIKey)
			if err != nil {
				logger.WithError(err).Error("failed to encrypt api key")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			userExchange.APIKeyHash = cipherText
		}

		if payload.APISecret != "" {
			cipherText, err := security.EncryptString(payload.APISecret)
			if err != nil {
				logger.WithError(err).Error("failed to encrypt api secret")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			userExchange.APISecretHash = cipherText
		}

		if payload.APIPassphrase != "" {
			cipherText, err := security.EncryptString(payload.APIPassphrase)
			if err != nil {
				logger.WithError(err).Error("failed to encrypt api passphrase")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			userExchange.APIPassphraseHash = cipherText
		}

		userExchange.ShowInForms = payload.ShowInForms

		if err := repository.GetUserExchangeStore().SaveUserExchange(userExchange); err != nil {
			logger.WithError(err).Error("failed to upsert user exchange")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		userExchange.Exchange = exchange

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(model.NewUserExchangeResponse(userExchange)); err != nil {
			logger.WithError(err).Error("failed to encode user exchange response")
		}
	}
}

func ListFormUserExchangesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok || user == nil {
			logger.Warn("user not found in context while listing form exchanges")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userExchanges, err := repository.GetUserExchangeStore().ListFormUserExchanges(user.ID)
		if err != nil {
			logger.WithError(err).Error("failed to list user exchanges for forms")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		responses := make([]model.UserExchangeResponse, 0, len(userExchanges))
		for i := range userExchanges {
			responses = append(responses, model.NewUserExchangeResponse(&userExchanges[i]))
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(responses); err != nil {
			logger.WithError(err).Error("failed to encode user exchange list response")
		}
	}
}

func DeleteUserExchangeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok || user == nil {
			logger.Warn("user not found in context while deleting user exchange")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		exchangeIDStr := chi.URLParam(r, "exchangeID")
		if exchangeIDStr == "" {
			http.Error(w, "exchangeID is required", http.StatusBadRequest)
			return
		}

		exchangeID, err := strconv.ParseUint(exchangeIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid exchangeID", http.StatusBadRequest)
			return
		}

		deleted, err := repository.GetUserExchangeStore().DeleteUserExchange(user.ID, uint(exchangeID))
		if err != nil {
			logger.WithError(err).Error("failed to delete user exchange")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !deleted {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func TestMexcConnectionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok || user == nil {
			logger.Warn("user not found in context while testing MEXC connection")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		exchangeIDStr := chi.URLParam(r, "exchangeID")
		if exchangeIDStr == "" {
			http.Error(w, "exchangeID is required", http.StatusBadRequest)
			return
		}

		exchangeID, err := strconv.ParseUint(exchangeIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid exchangeID", http.StatusBadRequest)
			return
		}

		userExchange, err := repository.GetUserExchangeStore().FindUserExchange(user.ID, uint(exchangeID))
		if err != nil {
			if errors.Is(err, repository.ErrUserExchangeNotFound) {
				http.Error(w, "user exchange not found", http.StatusNotFound)
				return
			}

			logger.WithError(err).Error("failed to fetch user exchange for MEXC connection test")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		exchange := userExchange.Exchange
		if exchange == nil {
			exchange, err = repository.GetUserExchangeStore().GetExchangeByID(uint(exchangeID))
			if err != nil {
				if errors.Is(err, repository.ErrExchangeNotFound) {
					http.Error(w, "exchange not found", http.StatusNotFound)
					return
				}

				logger.WithError(err).Error("failed to load exchange during MEXC connection test")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		if exchange == nil || !strings.EqualFold(exchange.Name, "mexc") {
			http.Error(w, "exchange is not MEXC", http.StatusBadRequest)
			return
		}

		if userExchange.APIKeyHash == "" || userExchange.APISecretHash == "" {
			http.Error(w, "missing API credentials for MEXC", http.StatusBadRequest)
			return
		}

		//apiKey, err := security.DecryptString(userExchange.APIKeyHash)
		//if err != nil {
		//	logger.WithError(err).Error("failed to decrypt api key")
		//	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		//	return
		//}
		//
		//apiSecret, err := security.DecryptString(userExchange.APISecretHash)
		//if err != nil {
		//	logger.WithError(err).Error("failed to decrypt api secret")
		//	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		//	return
		//}

		//connector := mexcConnectorFactory(apiKey, apiSecret)
		//if connector == nil {
		//	logger.Error("failed to build MEXC connector")
		//	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		//	return
		//}
		//
		//if err := connector.TestConnection(); err != nil {
		//	logger.WithError(err).Warn("MEXC connection test failed")
		//	http.Error(w, "Failed to connect to MEXC", http.StatusBadGateway)
		//	return
		//}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			logger.WithError(err).Error("failed to encode MEXC test response")
		}
	}
}
