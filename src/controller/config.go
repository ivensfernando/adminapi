package controller

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	OrderSizePercent int `envconfig:"ORDER_SIZE_PERCENT" default:"25"`
}

func GetConfig() Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}
