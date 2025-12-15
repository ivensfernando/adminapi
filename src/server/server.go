package server

import (
	"adminapi/src/auth"
	"adminapi/src/handler"
	"adminapi/src/lookup"
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	logger "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func StartServer(port string) {
	// Router with middleware
	r := chi.NewRouter()
	// === Global Middleware ===

	r.Use(auth.CorsHandler())

	r.Use(requestLogger())
	r.Use(sharedSecretAuth()) // <- Our custom auth middleware

	// Public routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.WithError(err).Error(" \"/health error")
		}
	})

	r.Post("/trading/webhook/{token}", handler.AlertHandler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/lookup", func(r chi.Router) {
			r.Get("/exchanges", lookup.ListExchanges())
			r.Get("/pairs", lookup.ListPairs())
		})

		r.Post("/auth/register", handler.RegisterHandler())
		r.Post("/auth/login", handler.LoginHandler())

		// Protected routes (JWT required)
		r.Group(func(r chi.Router) {

			r.Use(auth.RequireAuthMiddleware()) // ✅ <— protect the routes

			r.Get("/me", handler.MeHandler())
			r.Put("/me", handler.UpdateUserHandler())
			r.Get("/logout", handler.LogoutHandler())

			r.Route("/user-exchanges", func(r chi.Router) {
				r.Post("/", handler.UpsertUserExchangeHandler())
				r.Get("/forms", handler.ListFormUserExchangesHandler())
				r.Post("/{exchangeID}/test", handler.TestMexcConnectionHandler())
				r.Delete("/{exchangeID}", handler.DeleteUserExchangeHandler())
			})

			r.Post("/webhooks", handler.CreateWebhookHandler())
			r.Get("/webhooks", handler.ListWebhooksHandler())
			r.Put("/webhooks/{id}", handler.UpdateWebhookHandler())
			r.Delete("/webhooks/{id}", handler.DeleteWebhookHandler())
			r.Get("/webhook-alerts", handler.ListWebhookAlertsHandler())

		})
	})
	// Graceful server
	// Server setup
	addr := ":" + port
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Fatal("Server crashed")
		}
	}()

	// Shutdown on SIGINT or SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Shutdown error")
	}
}
