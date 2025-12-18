package ohlcvcrypto

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	StartDt              time.Time `envconfig:"START_DATE" default:"2025-12-18T00:00:00Z"`
	EndDt                time.Time `envconfig:"END_DATE" default:"2027-01-31T00:00:00Z"`
	DurationStr          string    `envconfig:"DURATION" default:"1h"`
	SupportedResolutions string    `envconfig:"SUPPORTED_RESOLUTIONS" default:"1,5,15,30,60,1h,4h,1D,1W,1M"`
	AutoMode             bool      `envconfig:"AUTO_MODE" default:"false"`
	Symbol               string    `envconfig:"SYMBOL" default:"BTC"`
	Quote                string    `envconfig:"QUOTE" default:"USDT"`
	Limit                int       `envconfig:"LIMIT" default:"1000"`
}

func GetConfig() *Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return &config
}
