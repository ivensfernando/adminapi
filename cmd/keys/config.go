package keys

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ExchangeID  uint `envconfig:"EXCHANGE_ID" default:"1"`
	RunOnServer bool `envconfig:"RUN_ON_SERVER" default:"true"`
}

func GetConfig() Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}
