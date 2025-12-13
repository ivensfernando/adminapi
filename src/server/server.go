package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"vsC1Y2025V01/src/auth"
	"vsC1Y2025V01/src/handler"
	"vsC1Y2025V01/src/lookup"
)

func StartServer(port string, logger *logrus.Entry) {
	// Router with middleware
	r := chi.NewRouter()
	// === Global Middleware ===

	//r.Use(auth.AllowOriginMiddleware(logger), auth.OptionsMiddleware(logger))
	r.Use(auth.CorsHandler(logger))

	//r.Use(cors.Handler(cors.Options{
	//	AllowedOrigins:   []string{"http://localhost:3000"}, // React/React-Admin frontend
	//	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	//	AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
	//	AllowCredentials: true,
	//	MaxAge:           300,
	//}))
	r.Use(requestLogger(logger))
	r.Use(sharedSecretAuth(logger)) // <- Our custom auth middleware

	//r.Method("OPTIONS", "/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	//	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	//	w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Range")
	//	w.Header().Set("Access-Control-Allow-Credentials", "true")
	//	w.WriteHeader(http.StatusOK)
	//}))

	// Public routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.WithError(err).Error(" \"/health error")
		}
	})

	r.Post("/trading/webhook/{token}", handler.AlertHandler(logger))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/lookup", func(r chi.Router) {
			r.Get("/exchanges", lookup.ListExchanges(logger))
			r.Get("/pairs", lookup.ListPairs(logger))
		})

		r.Post("/auth/register", handler.RegisterHandler(logger))
		r.Post("/auth/login", handler.LoginHandler(logger))

		// Protected routes (JWT required)
		r.Group(func(r chi.Router) {

			r.Use(auth.RequireAuthMiddleware(logger)) // ✅ <— protect the routes

			r.Get("/me", handler.MeHandler(logger))
			r.Put("/me", handler.UpdateUserHandler(logger))
			r.Get("/logout", handler.LogoutHandler(logger))

			// CRUD Routes for Trades
			r.Get("/trades", handler.ListTradesHandler(logger))
			r.Get("/trades/{id}", handler.GetTradeHandler(logger))
			r.Post("/trades", handler.CreateTradeHandler(logger))
			r.Put("/trades/{id}", handler.UpdateTradeHandler(logger))
			r.Delete("/trades", handler.DeleteManyTradesHandler(logger))
			r.Delete("/trades/{id}", handler.DeleteTradeHandler(logger))

			r.Route("/user-exchanges", func(r chi.Router) {
				r.Post("/", handler.UpsertUserExchangeHandler(logger))
				r.Get("/forms", handler.ListFormUserExchangesHandler(logger))
				r.Post("/{exchangeID}/test", handler.TestMexcConnectionHandler(logger))
				r.Delete("/{exchangeID}", handler.DeleteUserExchangeHandler(logger))
			})

			r.Post("/webhooks", handler.CreateWebhookHandler(logger))
			r.Get("/webhooks", handler.ListWebhooksHandler(logger))
			r.Put("/webhooks/{id}", handler.UpdateWebhookHandler(logger))
			r.Delete("/webhooks/{id}", handler.DeleteWebhookHandler(logger))
			r.Get("/webhook-alerts", handler.ListWebhookAlertsHandler(logger))

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
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
