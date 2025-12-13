package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"vsC1Y2025V01/src/auth"
	"vsC1Y2025V01/src/model"
	"vsC1Y2025V01/src/payloads"
	"vsC1Y2025V01/src/repository"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload payloads.AuthPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
		if err != nil {
			logger.WithError(err).Error("Hashing failed")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		user := model.User{
			Username:    payload.Username,
			Password:    string(hash),
			Email:       payload.Email,
			FirstName:   payload.FirstName,
			LastName:    payload.LastName,
			Bio:         payload.Bio,
			AvatarURL:   payload.AvatarURL,
			PhoneNumber: payload.PhoneNumber,
			Timezone:    payload.Timezone,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := repository.GetUserRepository().Create(&user); err != nil {
			if errors.Is(err, repository.ErrUserAlreadyExists) {
				logger.WithError(err).Warn("User already exists")
				http.Error(w, "User already exists", http.StatusConflict)
				return
			}
			logger.WithError(err).Error("User registration failed")
			http.Error(w, "Registration error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		//w.WriteHeader(http.StatusOK)
		w.WriteHeader(http.StatusCreated)
		//w.Write([]byte("User created successfully"))
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func LoginHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Login attempt received")

		var payload payloads.AuthPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			logger.WithError(err).Warn("Failed to decode login payload")
			http.Error(w, "Invalid login", http.StatusBadRequest)
			return
		}

		logger.WithField("username", payload.Username).Info("Looking up user")

		user, err := repository.GetUserRepository().FindByUsername(payload.Username)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				logger.WithError(err).Warn("User not found or DB error")
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			logger.WithError(err).Error("Failed to lookup user for login")
			logger.WithError(err).Warn("User not found or DB error")
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password)); err != nil {
			logger.WithField("username", payload.Username).Warn("Password mismatch")
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		logger.WithField("user_id", user.ID).Info("Login successful, updating timestamps")

		// Update login times
		user.LastLogin = time.Now()
		user.LastSeen = time.Now()
		if err := repository.GetUserRepository().Update(user); err != nil {
			logger.WithError(err).Error("Failed to update last login timestamps")
		}

		// Generate JWT token
		token, err := auth.GenerateToken(user.ID)
		if err != nil {
			logger.WithError(err).Error("Failed to generate token")
			http.Error(w, "Token error", http.StatusInternalServerError)
			return
		}

		logger.WithField("user_id", user.ID).Info("Token generated, sending response")

		//w.Header().Set("Content-Type", "application/json")
		//json.NewEncoder(w).Encode(map[string]string{"token": token})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true, // ⚠️ Only works on HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   3600, // 1 hour
		})

		//w.WriteHeader(http.StatusOK)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
func LogoutHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("User logging out")

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Logged out"))
	}
}

//const UserKeyM contextKey = "user"

func MeHandler(logger *logrus.Entry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//user, ok := r.Context().Value("user").(*model.User)
		user, ok := r.Context().Value(auth.UserKey).(*model.User)
		if !ok || user == nil {
			logger.Warn("No user found in context1")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user.ToResponse())
	}
}
