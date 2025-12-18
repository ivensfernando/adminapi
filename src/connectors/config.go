package connectors

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HydraInstrumentID int     `envconfig:"HYDRA_INSTRUMENT_ID" default:"9910"`
	HydraSymbol       string  `envconfig:"HYDRA_SYMBOL" default:"BTC/USD.crypto"`
	HydraQTD          float64 `envconfig:"HYDRA_QTD" default:"0.00001"`
	HydraSLPercent    float64 `envconfig:"HYDRA_SL_PERCENT" default:"5"`

	KrakenQTD       float64 `envconfig:"KRAKEN_QTD" default:"0.0001"`
	KrakenSLPercent float64 `envconfig:"KRAKEN_SL_PERCENT" default:"5"`
	KrakenSymbol    string  `envconfig:"KRAKEN_SYMBOL" default:"PF_XBTUSD"`
}

func GetConfig() Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}
