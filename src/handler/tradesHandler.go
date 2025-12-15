package handler

//import (
//	"adminapi/src/auth"
//	"adminapi/src/database"
//	"adminapi/src/model"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"github.com/go-chi/chi/v5"
//	logger "github.com/sirupsen/logrus"
//	"net/http"
//	"strconv"
//	"strings"
//	"time"
//)
//
//type TradeListResponse struct {
//	Data  []model.Trade `json:"data"`
//	Total int           `json:"total"`
//}
//
////	func ListTradesHandler(logger *logrus.Entry) http.HandlerFunc {
////		return func(w http.ResponseWriter, r *http.Request) {
////			var trades []model.Trade
////			if err := database.DB.Find(&trades).Error; err != nil {
////				logger.WithError(err).Error("Failed to fetch trades")
////				http.Error(w, "Failed to fetch trades", http.StatusInternalServerError)
////				return
////			}
////			total := len(trades) // ideally you'd do a COUNT(*) query
////
////			resp := TradeListResponse{
////				Data:  trades,
////				Total: total,
////			}
////
////			w.Header().Set("Content-Type", "application/json")
////			json.NewEncoder(w).Encode(resp)
////		}
////	}
//
//func ListTradesHandler() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		//user, ok := auth.GetUserFromContext(r.Context())
//	//	if !ok {
//	//		http.Error(w, "Unauthorized", http.StatusUnauthorized)
//	//		return
//	//	}
//	//
//	//	offset, limit := ParseRangeParams(r)
//	//	sortField, sortDir := ParseSortParams(r)
//	//	filters := ParseFilterParams(r)
//	//
//	//	query := db.DB.Model(&model.Trade{}).Where("user_id = ?", user.ID)
//	//
//	//	// Apply filters
//	//	for k, v := range filters {
//	//		query = query.Where(fmt.Sprintf("%s = ?", k), v)
//	//	}
//	//
//	//	var trades []model.Trade
//	//	if err := query.Order(fmt.Sprintf("%s %s", sortField, sortDir)).
//	//		Offset(offset).Limit(limit).
//	//		Find(&trades).Error; err != nil {
//	//		logger.WithError(err).Error("Failed to fetch trades")
//	//		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//	//		return
//	//	}
//	//
//	//	var total int64
//	//	if err := query.Count(&total).Error; err != nil {
//	//		logger.WithError(err).Error("Failed to count trades")
//	//		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//	//		return
//	//	}
//	//
//	//	w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
//	//	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
//	//	w.Header().Set("Content-Type", "application/json")
//	//	w.WriteHeader(http.StatusOK)
//	//
//	//	if err := json.NewEncoder(w).Encode(trades); err != nil {
//	//		logger.WithError(err).Error("Failed to encode trades response")
//	//		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//	//		return
//	//	}
//	//}
//}
//
//func validateTradePayload(p model.TradePayload) error {
//	// required basics
//	if strings.TrimSpace(p.Symbol) == "" {
//		return errors.New("symbol is required")
//	}
//	if p.ContractType == nil || strings.TrimSpace(*p.ContractType) == "" {
//		return errors.New("symbol is required")
//	}
//	if strings.TrimSpace(p.Type) == "" {
//		//return errors.New("type is required")
//	}
//	//if p.Price <= 0 {
//	//	return errors.New("price must be > 0")
//	//}
//	//if p.Quantity <= 0 {
//	//	return errors.New("size (quantity) must be > 0")
//	//}
//	// exchange allow-list (optional field)
//	//if p.Exchange != nil {
//	//	ok := map[string]bool{"Binance": true, "Kucoin": true, "Mexc": true}
//	//	if !ok[*p.Exchange] {
//	//		return fmt.Errorf("exchange must be one of: Binance, Kucoin, Mexc")
//	//	}
//	//}
//	// side consistency
//	if p.IsShort == p.IsLong {
//		return errors.New("exactly one of isShort or isLong must be true")
//	}
//	// take profit coupling
//	if p.TakeProfitEnabled && (p.TakeProfit == nil || *p.TakeProfit <= 0) {
//		return errors.New("takeProfit must be provided and > 0 when takeProfitEnabled is true")
//	}
//	// leverage sanity (if provided)
//	//if p.Leverage != nil && *p.Leverage <= 0 {
//	//	return errors.New("leverage must be > 0 when provided")
//	//}
//	return nil
//}
//
//func parseTradeDate(dateStr, timeStr string, loc *time.Location) (time.Time, error) {
//	// Try RFC3339 first
//	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
//		return t, nil
//	}
//	// Fallback: combine date + time (assumed local timezone)
//	if strings.TrimSpace(dateStr) == "" {
//		return time.Time{}, errors.New("tradeDate is required")
//	}
//	if strings.TrimSpace(timeStr) == "" {
//		timeStr = "00:00"
//	}
//	if loc == nil {
//		loc = time.Local
//	}
//	d, err := time.ParseInLocation("2006-01-02", dateStr, loc)
//	if err != nil {
//		return time.Time{}, fmt.Errorf("invalid tradeDate: %w", err)
//	}
//	tt, err := time.ParseInLocation("15:04", timeStr, loc)
//	if err != nil {
//		return time.Time{}, fmt.Errorf("invalid tradeTime: %w", err)
//	}
//	return time.Date(d.Year(), d.Month(), d.Day(), tt.Hour(), tt.Minute(), 0, 0, loc), nil
//}
//
//func CreateTrade(user model.User, payload model.TradePayload, loc *time.Location) (*model.Trade, error) {
//	if err := validateTradePayload(payload); err != nil {
//		return nil, err
//	}
//
//	parsedDate, err := parseTradeDate(payload.TradeDate, payload.TradeTime, loc)
//	if err != nil {
//		return nil, err
//	}
//
//	if payload.IsLong {
//		payload.Type = "Buy/Long"
//	} else {
//		payload.Type = "Sell/Short"
//	}
//
//	trade := model.Trade{
//		Exchange:          payload.Exchange,
//		Symbol:            payload.Symbol,
//		TradeDate:         parsedDate,
//		TradeTime:         payload.TradeTime,
//		ContractType:      payload.ContractType,
//		MarginMode:        payload.MarginMode,
//		Leverage:          payload.Leverage,
//		AssetMode:         payload.AssetMode,
//		OrderType:         payload.OrderType,
//		Price:             payload.Price,
//		Quantity:          payload.Quantity, // keep as float64
//		StopPrice:         payload.StopPrice,
//		TakeProfitEnabled: payload.TakeProfitEnabled,
//		ReduceOnly:        payload.ReduceOnly,
//		StopLoss:          payload.StopLoss,
//		TakeProfit:        payload.TakeProfit,
//		IsShort:           payload.IsShort, // fixed typo
//		IsLong:            payload.IsLong,
//
//		Type:       payload.Type,
//		EntryPrice: payload.EntryPrice,
//		ExitPrice:  payload.ExitPrice,
//		Fee:        payload.Fee,
//		Indicators: payload.Indicators,
//		Sentiment:  payload.Sentiment,
//		Notes:      payload.Notes,
//
//		UserID: user.ID,
//		// CreatedAt / UpdatedAt are auto-managed by GORM if you omit them
//	}
//
//	// Enforce consistency: if TP disabled, ignore provided value
//	if !trade.TakeProfitEnabled {
//		trade.TakeProfit = nil
//	}
//
//	if err := db.DB.Create(&trade).Error; err != nil {
//		return nil, err
//	}
//
//	return &trade, nil
//}
//
//func GetTradeHandler() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//
//		idStr := chi.URLParam(r, "id")
//		if idStr == "" {
//			logger.WithFields(map[string]interface{}{"id": idStr}).Error("Missing trade ID")
//			http.Error(w, "Missing trade ID", http.StatusBadRequest)
//			return
//		}
//		logger.WithFields(map[string]interface{}{"id": idStr}).Info("Getting trade with id")
//
//		id, err := strconv.Atoi(idStr)
//		if err != nil {
//			http.Error(w, "Invalid trade ID", http.StatusBadRequest)
//			return
//		}
//
//		var trade model.Trade
//		if err := db.DB.First(&trade, id).Error; err != nil {
//			logger.WithError(err).Error("Trade not found")
//			http.Error(w, "Trade not found", http.StatusNotFound)
//			return
//		}
//
//		tradeR := model.TradeResponse{
//			ID:         trade.ID,
//			Symbol:     trade.Symbol,
//			TradeDate:  trade.TradeDate.Format("2006-01-02"),
//			Type:       trade.Type,
//			Leverage:   trade.Leverage,
//			EntryPrice: trade.EntryPrice,
//
//			ExitPrice:  trade.ExitPrice,
//			Fee:        trade.Fee,
//			Indicators: trade.Indicators,
//			Sentiment:  trade.Sentiment,
//			StopLoss:   trade.StopLoss,
//			TakeProfit: trade.TakeProfit,
//			Exchange:   trade.Exchange,
//		}
//
//		w.Header().Set("Content-Type", "application/json")
//		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
//		w.Header().Set("Access-Control-Expose-Headers", "Content-Range,X-Total-Count")
//		w.Header().Set("X-Total-Count", "1")
//		w.WriteHeader(http.StatusOK)
//		if err := json.NewEncoder(w).Encode(map[string]interface{}{
//			"data": tradeR,
//		}); err != nil {
//			logger.WithError(err).Error("Failed to encode trade response")
//			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//			return
//		}
//	}
//}
//
////type contextKey string
//
////const UserKey contextKey = "user"
//
//func CreateTradeHandler() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		var payload model.TradePayload
//		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
//			logger.WithError(err).Warn("Invalid trade payload")
//			http.Error(w, "Bad Request", http.StatusBadRequest)
//			return
//		}
//
//		//parsedDate, err := time.Parse("2006-01-02", payload.TradeDate)
//		_, err := time.Parse(time.RFC3339, payload.TradeDate)
//		if err != nil {
//			logger.WithError(err).Warn("Invalid date format")
//			http.Error(w, "Invalid date format", http.StatusBadRequest)
//			return
//		}
//
//		//user, ok := r.Context().Value(UserKey).(*model.User)
//		userPayload, ok := auth.GetUserFromContext(r.Context())
//		if !ok || userPayload == nil {
//			logger.Warn("No user found in context")
//			http.Error(w, "Unauthorized", http.StatusUnauthorized)
//			return
//		}
//
//		loc := new(time.Location)
//
//		var user model.User
//		if err := db.DB.Where("username = ?", userPayload.Username).First(&user).Error; err != nil {
//			logger.WithError(err).Warn("User not found or DB error")
//			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
//			return
//		}
//
//		trade, err := CreateTrade(user, payload, loc)
//		if err != nil {
//			logger.WithError(err).Warn("Invalid trade payload")
//			http.Error(w, "Bad Request", http.StatusBadRequest)
//			return
//		}
//
//		//trade.CreatedAt = time.Now()
//		//trade.UpdatedAt = time.Now()
//
//		//if err := database.DB.Create(&trade).Error; err != nil {
//		//	logger.WithError(err).Error("Failed to create trade")
//		//	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//		//	return
//		//}
//
//		//w.WriteHeader(http.StatusCreated)
//		//json.NewEncoder(w).Encode(map[string]interface{}{
//		//	"data": map[string]interface{}{
//		//		"id":          trade.ID,
//		//		"symbol":      trade.Symbol,
//		//		"trade_date":  trade.TradeDate,
//		//		"type":        trade.Type,
//		//		"leverage":    trade.Leverage,
//		//		"entry_price": trade.EntryPrice,
//		//		"exit_price":  trade.ExitPrice,
//		//		"fee":         trade.Fee,
//		//		"indicators":  trade.Indicators,
//		//		"sentiment":   trade.Sentiment,
//		//		"stop_loss":   trade.StopLoss,
//		//		"take_profit": trade.TakeProfit,
//		//		"exchange":    trade.Exchange,
//		//		"user_id":     trade.UserID,
//		//		"created_at":  trade.CreatedAt,
//		//		"updated_at":  trade.UpdatedAt,
//		//	},
//		//})
//		w.WriteHeader(http.StatusCreated)
//		if err := json.NewEncoder(w).Encode(map[string]interface{}{
//			"data": trade,
//		}); err != nil {
//			logger.WithError(err).Error("Failed to encode trade response")
//			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//			return
//		}
//	}
//}
//
////func ApplyTradePatch(t *model.Trade, p model.UpdateTradePayload, loc *time.Location) error {
////	if p.Exchange != nil {
////		t.Exchange = *p.Exchange
////	}
////	if p.Symbol != nil {
////		t.Symbol = *p.Symbol
////	}
////	if p.TradeDate != nil || p.TradeTime != nil {
////		dateStr := t.TradeDate.Format("2006-01-02")
////		timeStr := t.TradeTime
////		//if p.TradeDate != nil && *p.TradeDate != nil {
////		//	dateStr = **p.TradeDate
////		//}
////		if p.TradeTime != nil {
////			//timeStr = derefStringOr(t.TradeTime, p.TradeTime)
////		}
////		tt, err := parseTradeDate(dateStr, timeStr, loc)
////		if err != nil {
////			return err
////		}
////		t.TradeDate, t.TradeTime = tt, timeStr
////	}
////	if p.MarginMode != nil {
////		t.MarginMode = *p.MarginMode
////	}
////	if p.Leverage != nil {
////		t.Leverage = *p.Leverage
////	}
////	if p.AssetMode != nil {
////		t.AssetMode = *p.AssetMode
////	}
////	if p.OrderType != nil {
////		t.OrderType = *p.OrderType
////	}
////	if p.Price != nil {
////		t.Price = *p.Price
////	}
////	if p.Quantity != nil {
////		t.Quantity = *p.Quantity
////	}
////	if p.StopPrice != nil {
////		t.StopPrice = *p.StopPrice
////	}
////	if p.TakeProfitEnabled != nil {
////		t.TakeProfitEnabled = *p.TakeProfitEnabled
////	}
////	if p.ReduceOnly != nil {
////		t.ReduceOnly = *p.ReduceOnly
////	}
////	if p.TakeProfit != nil {
////		t.TakeProfit = *p.TakeProfit
////	}
////	if p.StopLoss != nil {
////		t.StopLoss = *p.StopLoss
////	}
////	if p.IsShort != nil {
////		t.IsShort = *p.IsShort
////	}
////	if p.IsLong != nil {
////		t.IsLong = *p.IsLong
////	}
////	if p.Notes != nil {
////		t.Notes = *p.Notes
////	}
////	if p.Type != nil {
////		t.Type = *p.Type
////	}
////	if p.EntryPrice != nil {
////		t.EntryPrice = *p.EntryPrice
////	}
////	if p.ExitPrice != nil {
////		t.ExitPrice = *p.ExitPrice
////	}
////	if p.Fee != nil {
////		t.Fee = *p.Fee
////	}
////	if p.Indicators != nil {
////		t.Indicators = *p.Indicators
////	}
////	if p.Sentiment != nil {
////		t.Sentiment = *p.Sentiment
////	}
////
////	// Re-validate invariants after patch:
////	if t.IsShort == t.IsLong {
////		return errors.New("exactly one of isShort or isLong must be true")
////	}
////	if t.TakeProfitEnabled && (t.TakeProfit == nil || *t.TakeProfit <= 0) {
////		return errors.New("takeProfit must be provided and > 0 when enabled")
////	}
////	return nil
////}
//
//func UpdateTradeHandler() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		idStr := chi.URLParam(r, "id")
//		if idStr == "" {
//			logger.WithFields(map[string]interface{}{"id": idStr}).Error("Missing trade ID")
//			http.Error(w, "Missing trade ID", http.StatusBadRequest)
//			return
//		}
//		logger.WithFields(map[string]interface{}{"id": idStr}).Info("Getting trade with id")
//
//		id, err := strconv.Atoi(idStr)
//		if err != nil {
//			http.Error(w, "Invalid trade ID", http.StatusBadRequest)
//			return
//		}
//
//		var trade model.Trade
//		if err := db.DB.First(&trade, id).Error; err != nil {
//			logger.WithError(err).Error("Trade not found")
//			http.Error(w, "Trade not found", http.StatusNotFound)
//			return
//		}
//
//		var payload model.TradePayload
//		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
//			logger.WithError(err).Warn("Invalid trade payload")
//			http.Error(w, "Bad Request", http.StatusBadRequest)
//			return
//		}
//
//		parsedDate, err := time.Parse("2006-01-02", payload.TradeDate)
//		if err != nil {
//			logger.WithError(err).Warn("Invalid date format")
//			http.Error(w, "Invalid date format", http.StatusBadRequest)
//			return
//		}
//
//		trade.Symbol = payload.Symbol
//		trade.TradeDate = parsedDate // ✅ now it' a time.Time!
//		trade.Type = payload.Type
//		trade.Leverage = payload.Leverage
//		trade.EntryPrice = payload.EntryPrice
//		trade.ExitPrice = payload.ExitPrice
//		trade.Fee = payload.Fee
//		trade.Indicators = payload.Indicators
//		trade.Sentiment = payload.Sentiment
//		trade.StopLoss = payload.StopLoss
//		trade.TakeProfit = payload.TakeProfit
//		trade.Exchange = payload.Exchange
//
//		trade.UpdatedAt = time.Now()
//
//		if err := db.DB.Save(&trade).Error; err != nil {
//			logger.WithError(err).Error("Failed to update trade")
//			http.Error(w, "Failed to update trade", http.StatusInternalServerError)
//			return
//		}
//
//		w.Header().Set("Content-Type", "application/json")
//		w.WriteHeader(http.StatusOK)
//		if err := json.NewEncoder(w).Encode(map[string]interface{}{
//			"data": trade,
//		}); err != nil {
//			logger.WithError(err).Error("Failed to encode trade response")
//			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//			return
//		}
//	}
//}
//
//func DeleteTradeHandler() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		idStr := chi.URLParam(r, "id")
//		if idStr == "" {
//			logger.WithFields(map[string]interface{}{"id": idStr}).Error("Missing trade ID")
//			http.Error(w, "Missing trade ID", http.StatusBadRequest)
//			return
//		}
//		logger.WithFields(map[string]interface{}{"id": idStr}).Info("Getting trade with id")
//
//		id, err := strconv.Atoi(idStr)
//		if err != nil {
//			http.Error(w, "Invalid trade ID", http.StatusBadRequest)
//			return
//		}
//
//		var trade model.Trade
//		if err := db.DB.First(&trade, id).Error; err != nil {
//			logger.WithError(err).Error("Trade not found")
//			http.Error(w, "Trade not found", http.StatusNotFound)
//			return
//		}
//
//		if err := db.DB.Delete(&trade).Error; err != nil {
//			logger.WithError(err).Error("Failed to delete trade")
//			http.Error(w, "Failed to delete trade", http.StatusInternalServerError)
//			return
//		}
//
//		w.WriteHeader(http.StatusNoContent)
//	}
//}
//
//func DeleteManyTradesHandler() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		var payload struct {
//			IDs []uint `json:"id"`
//		}
//
//		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
//			logger.WithError(err).Warn("Invalid payload for deleteMany")
//			http.Error(w, "Invalid request", http.StatusBadRequest)
//			return
//		}
//
//		if len(payload.IDs) == 0 {
//			http.Error(w, "No IDs provided", http.StatusBadRequest)
//			return
//		}
//
//		if err := db.DB.Delete(&model.Trade{}, payload.IDs).Error; err != nil {
//			logger.WithError(err).Error("Failed to delete trades")
//			http.Error(w, "Failed to delete trades", http.StatusInternalServerError)
//			return
//		}
//
//		logger.WithField("ids", payload.IDs).Info("Deleted trades")
//
//		w.Header().Set("Content-Type", "application/json")
//		if err := json.NewEncoder(w).Encode(map[string]interface{}{
//			"data": payload.IDs,
//		}); err != nil {
//			logger.WithError(err).Error("Failed to encode delete response")
//			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//			return
//		}
//	}
//}
