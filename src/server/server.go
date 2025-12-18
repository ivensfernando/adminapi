package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	logger "github.com/sirupsen/logrus"
)

func StartServer(port string) {
	// Router with middleware
	r := chi.NewRouter()
	// === Global Middleware ===

	// Public routes
	r.Get("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.WithError(err).Error(" \"/health error")
		}
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
