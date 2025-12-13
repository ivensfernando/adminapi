package auth

import (
	"adminapi/src/repository"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/cors"
	logger "github.com/sirupsen/logrus"
)

//type contextKey string
//
//const UserKey contextKey = "user"

func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.WithField("path", r.URL.Path).Info("Authenticating request")

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				logger.Warn("Missing or malformed Authorization header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			logger.Debug("Bearer token extracted")

			userID, err := ParseToken(token)
			if err != nil {
				logger.WithError(err).Warn("Invalid or expired token")
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			logger.WithField("user_id", userID).Debug("Token parsed successfully")

			user, err := repository.GetUserRepository().FindByID(userID)
			if err != nil {
				if errors.Is(err, repository.ErrUserNotFound) {
					logger.WithError(err).Warn("User not found in database")
				} else {
					logger.WithError(err).Error("Failed to load user for auth middleware")
				}
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}
			logger.WithField("username", user.Username).Info("User authenticated")

			// Update last seen timestamp
			user.LastSeen = time.Now()
			if err := repository.GetUserRepository().Update(user); err != nil {
				logger.WithError(err).Error("Failed to update last seen")
			} else {
				logger.WithField("username", user.Username).Debug("Last seen updated")
			}

			ctx := context.WithValue(r.Context(), UserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//if r.Method == http.MethodOptions {
			//	next.ServeHTTP(w, r)
			//	return
			//}
			logger.WithField("path", r.URL.Path).Info("Authenticating via cookie")

			cookie, err := r.Cookie("token")
			if err != nil {
				logger.Warn("Missing token cookie")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := ParseToken(cookie.Value)
			if err != nil {
				logger.WithError(err).Warn("Invalid token in cookie")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := repository.GetUserRepository().FindByID(userID)
			if err != nil {
				if errors.Is(err, repository.ErrUserNotFound) {
					logger.WithError(err).Warn("User not found")
				} else {
					logger.WithError(err).Error("Failed to load user for cookie auth")
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user.LastSeen = time.Now()
			if err := repository.GetUserRepository().Update(user); err != nil {
				logger.WithError(err).Error("Failed to persist last seen timestamp")
			}

			ctx := context.WithValue(r.Context(), UserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AllowOrigin(w http.ResponseWriter, r *http.Request) {
	logger.WithField("url", r.URL).Debug("AllowOrigin, OPTIONS")

	w.Header().Set("Content-Type", "application/json")

	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, X-DEFYTRENDS-KEY, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, GET, POST, OPTIONS")

	//if origin := r.Header.Get("Origin"); origin != "" {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")

}

func AllowOriginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AllowOrigin(w, r)
		next.ServeHTTP(w, r)
	})
}

func OptionsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			logger.WithField("url", r.URL).Debug("OptionsMiddleware, OPTIONS")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CorsHandler() func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Range"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
