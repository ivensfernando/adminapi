package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"adminapi/src/auth"
	"adminapi/src/database"
	"adminapi/src/model"
	"adminapi/src/repository"

	logger "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func UpdateUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok || user == nil {
			logger.Warn("user not found in context during profile update")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var payload model.UpdateUserPayload
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			logger.WithError(err).Warn("invalid user update payload")
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if payload.Email != nil {
			user.Email = strings.TrimSpace(*payload.Email)
		}
		if payload.FirstName != nil {
			user.FirstName = strings.TrimSpace(*payload.FirstName)
		}
		if payload.LastName != nil {
			user.LastName = strings.TrimSpace(*payload.LastName)
		}
		if payload.Bio != nil {
			user.Bio = strings.TrimSpace(*payload.Bio)
		}
		if payload.AvatarURL != nil {
			user.AvatarURL = strings.TrimSpace(*payload.AvatarURL)
		}

		user.UpdatedAt = time.Now()

		if err := database.MainDB.Save(user).Error; err != nil {
			logger.WithError(err).Error("failed to update user profile")
			http.Error(w, "Unable to update profile", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(user.ToResponse()); err != nil {
			logger.WithError(err).Error("failed to encode user response")
		}
	}
}

func ChangePasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok || user == nil {
			logger.Warn("user not found in context during password change")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var payload model.ChangePasswordPayload
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			logger.WithError(err).Warn("invalid change password payload")
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if payload.CurrentPassword == "" || payload.NewPassword == "" {
			http.Error(w, "Current and new passwords are required", http.StatusBadRequest)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.CurrentPassword)); err != nil {
			logger.WithField("user_id", user.ID).Warn("current password mismatch")
			http.Error(w, "Invalid current password", http.StatusUnauthorized)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			logger.WithError(err).Error("failed to hash new password")
			http.Error(w, "Unable to update password", http.StatusInternalServerError)
			return
		}

		user.Password = string(hashedPassword)

		if err := repository.GetUserRepository().Update(user); err != nil {
			logger.WithError(err).Error("failed to update user password")
			http.Error(w, "Unable to update password", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "password updated"}); err != nil {
			logger.WithError(err).Error("failed to encode change password response")
		}
	}
}
