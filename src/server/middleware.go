package server

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"time"
)

// Logs every request with timing
func requestLogger(logger *logrus.Entry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.WithFields(logrus.Fields{
				"method": r.Method,
				"path":   r.URL.Path,
				"took":   time.Since(start),
			}).Info("Request")
		})
	}
}

// Auth check using header "X-Secret-Key"
func sharedSecretAuth(logger *logrus.Entry) func(http.Handler) http.Handler {
	secret := os.Getenv("SHARED_SECRET")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/trading/webhook/") {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get("X-Secret-Key") != secret {
				logger.Warn("Unauthorized: invalid secret")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
