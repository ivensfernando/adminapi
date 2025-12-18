package ohlcvcrypto

import (
	"database/sql"
	"errors"
	"net/http"
	common "strategyexecutor/src/model"
	"time"

	"github.com/shopspring/decimal"
	logger "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/nntaoli-project/goex"
	"github.com/nntaoli-project/goex/binance"
)

const (
	Duration1m = "1m"
	Duration1h = "1h"
)

type OHLCVCrypto struct {
	Log      *logger.Entry
	DB       *gorm.DB
	Config   *Config
	exchange goex.API
}

func (o *OHLCVCrypto) Start() error {
	o.Config = GetConfig()

	o.exchange = o.newBinanceInstance()

	if o.Config.AutoMode {
		if err := o.determineStartPoint(); err != nil {
			return err
		}
	}

	err := o.aggregateAndSave()

	return err
}

func (*OHLCVCrypto) newBinanceInstance() *binance.Binance {
	apiConfig := &goex.APIConfig{
		HttpClient: http.DefaultClient,
		Endpoint:   binance.GLOBAL_API_BASE_URL,
	}
	return binance.NewWithConfig(apiConfig)
}

func (o *OHLCVCrypto) aggregateAndSave() error {
	series, err := o.fetchOHLCVSeries()
	if err != nil {
		return err
	}

	for i := range series {
		result := series[i]

		var target interface{}
		target = &common.OHLCVBase{
			Datetime: time.Unix(result.Timestamp, 0).UTC(),
			Open:     decimal.NewFromFloat(result.Open),
			High:     decimal.NewFromFloat(result.High),
			Low:      decimal.NewFromFloat(result.Low),
			Close:    decimal.NewFromFloat(result.Close),
			Volume:   decimal.NewFromFloat(result.Vol),
			Symbol:   result.Pair.String(),
		}

		if o.Config.DurationStr == Duration1m {
			target = target.(*common.OHLCVBase).ConvertToOHLCVCrypto1m()
		} else if o.Config.DurationStr == Duration1h {
			target = target.(*common.OHLCVBase).ConvertToOHLCVCrypto1h()
		}

		// Upsert: on conflict on (datetime, symbol) do update
		if err := o.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "datetime"}, {Name: "symbol"}}, // Composite unique index columns
			DoUpdates: clause.AssignmentColumns([]string{"open", "high", "low", "close", "volume"}),
		}).Create(target).Error; err != nil {
			o.Log.WithError(err).Error("aggregateAndSave, Create, ")
			return err
		}

		o.Log.WithFields(logger.Fields{
			"Symbol":    o.Config.Symbol,
			"Price":     target,
			"Timestamp": time.Now().UTC(),
		}).Info("OHLCV data inserted or updated in database")
	}

	return nil
}

func (o *OHLCVCrypto) determineStartPoint() error {
	o.Config.StartDt = o.Config.StartDt.Add(-o.parseDuration())
	o.Config.EndDt = time.Now()

	var latestTime *sql.NullTime
	result := o.getModel().
		Select("MAX(datetime)").
		Where("symbol = ?", o.Config.Symbol+"_"+o.Config.Quote).
		Take(&latestTime)

	o.Log.
		WithField("latestTime", latestTime).
		Info("determineStartPoint")

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			o.Log.
				WithError(result.Error).
				WithField("StartDt", o.Config.StartDt.String()).
				WithField("EndDt", o.Config.EndDt.String()).
				Error("no records found, start from the configured StartDt")
		} else {
			o.Log.
				WithError(result.Error).
				Error("Failed to query latest datetime")
			return result.Error
		}
	}

	if latestTime.Valid {
		// If there's a valid latest time, set the start time one interval after the last recorded time
		o.Config.StartDt = latestTime.Time.Add(-o.parseDuration())
		o.Log.
			WithField("StartDt", o.Config.StartDt.String()).
			WithField("EndDt", o.Config.EndDt.String()).
			Info("determineStartPoint valid date found")
	} else {
		// If no valid time was returned, assume there is no data and start from the configured StartDt
		err := errors.New("no existing MAX(datetime) found")
		o.Log.
			WithError(err).
			WithField("StartDt", o.Config.StartDt.String()).
			WithField("EndDt", o.Config.EndDt.String()).
			Error("determineStartPoint invalid date found")
	}

	return nil
}

func (o *OHLCVCrypto) fetchOHLCVSeries() ([]goex.Kline, error) {
	targetSymbol := goex.NewCurrencyPair(goex.Currency{Symbol: o.Config.Symbol}, goex.Currency{Symbol: o.Config.Quote})

	const millis = 1000
	klines, err := o.exchange.GetKlineRecords(
		targetSymbol,
		o.parseDurationToGoex(),
		o.Config.Limit,
		goex.OptionalParameter{}.
			Optional("startTime", o.Config.StartDt.Unix()*millis).
			Optional("endTime", o.Config.EndDt.Unix()*millis),
	)
	if err != nil {
		return nil, err
	}

	return klines, nil
}

func (o *OHLCVCrypto) parseDuration() time.Duration {
	var duration time.Duration
	switch o.Config.DurationStr {
	case Duration1m:
		duration = time.Minute
	case Duration1h:
		duration = time.Hour
	default:
		panic("invalid DURATION env var")
	}
	return duration
}

func (o *OHLCVCrypto) parseDurationToGoex() goex.KlinePeriod {
	var duration goex.KlinePeriod
	switch o.Config.DurationStr {
	case Duration1m:
		duration = goex.KLINE_PERIOD_1MIN
	case Duration1h:
		duration = goex.KLINE_PERIOD_1H
	default:
		panic("invalid DURATION env var")
	}
	return duration
}

func (o *OHLCVCrypto) getModel() (tx *gorm.DB) {
	switch o.Config.DurationStr {
	case Duration1m:
		tx = o.DB.Model(&common.OHLCVCrypto1m{})
	case Duration1h:
		tx = o.DB.Model(&common.OHLCVCrypto1h{})
	default:
		panic("getModel, invalid DURATION")
	}
	return tx
}
