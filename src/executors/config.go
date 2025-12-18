package executors

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	APIKey         string        `envconfig:"PHEMEX_API_KEY"`
	APISecret      string        `envconfig:"PHEMEX_API_SECRET"`
	UserID         string        `envconfig:"USER_ID"`
	BaseURL        string        `envconfig:"BASE_URL" default:"https://testnet-api.phemex.com"`
	TargetExchange string        `envconfig:"TARGET_EXCHANGE" default:"phemex"`
	TargetSymbol   string        `envconfig:"TARGET_SYMBOL" default:"BTCUSD"`
	LoopPeriod     time.Duration `envconfig:"LOOP_PERIOD" default:"30s"`
}

func GetConfig() Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}
