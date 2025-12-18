package controller

import (
	"context"
	"encoding/json"
	logger "github.com/sirupsen/logrus"
	"runtime/debug"
	"strategyexecutor/src/model"
	"strategyexecutor/src/repository"
	"strings"
	"time"
)

// PercentOfFloatSafe returns the percentage of a float64 value using a safe clamped percent (1–100).
// If percent is out of range, it is automatically adjusted and logged.
func PercentOfFloatSafe(value float64, percent int) float64 {
	originalPercent := percent

	if percent < 1 {
		percent = 1
		logger.WithFields(map[string]interface{}{
			"value":        value,
			"original_pct": originalPercent,
			"adjusted_pct": percent,
		}).Warn("Percent below minimum, clamped to 1")
	}

	if percent > 100 {
		percent = 100
		logger.WithFields(map[string]interface{}{
			"value":        value,
			"original_pct": originalPercent,
			"adjusted_pct": percent,
		}).Warn("Percent above maximum, clamped to 100")
	}

	result := value * float64(percent) / 100.0

	logger.WithFields(map[string]interface{}{
		"value":   value,
		"percent": percent,
		"result":  result,
	}).Debug("Computed percentage of float value")

	return result
}

// NormalizeToUSDT ensures that a symbol ends with USDT.
// Examples:
//
//	BTCUSD  -> BTCUSDT
//	ETHUSD  -> ETHUSDT
//	BTCUSDT -> BTCUSDT
//	ethusd  -> ETHUSDT
func NormalizeToUSDT(symbol string) string {
	if symbol == "" {
		return symbol
	}

	s := strings.ToUpper(strings.TrimSpace(symbol))

	// If it already ends with USDT, nothing to do
	if strings.HasSuffix(s, "USDT") {
		return s
	}

	// If it ends with USD, replace with USDT
	if strings.HasSuffix(s, "USD") {
		base := strings.TrimSuffix(s, "USD")
		return base + "USDT"
	}

	// Otherwise, return as is (do not force)
	return s
}

// Capture records a system exception, logs it locally, and optionally
// persists it in the database.
func Capture(
	ctx context.Context,
	repo *repository.ExceptionRepository,
	service string,
	module string,
	method string,
	level string,
	err error,
	contextData map[string]interface{},
) {

	if err == nil {
		return
	}

	var ctxJSON string
	if contextData != nil {
		if b, e := json.Marshal(contextData); e == nil {
			ctxJSON = string(b)
		}
	}

	exc := &model.Exception{
		Service:   service,
		Module:    module,
		Method:    method,
		Message:   err.Error(),
		Stack:     string(debug.Stack()),
		Level:     level,
		Context:   ctxJSON,
		CreatedAt: time.Now(),
	}

	// Local log
	logger.WithFields(map[string]interface{}{
		"service": service,
		"module":  module,
		"method":  method,
		"level":   level,
	}).WithError(err).Error("System exception captured")

	// Persist in database
	if repo != nil {
		if e := repo.Create(ctx, exc); e != nil {
			logger.WithError(e).Error("Failed to persist exception")
		}
	}
}
