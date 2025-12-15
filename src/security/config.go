package security

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ExchangeCRKey string `envconfig:"EXCHANGE_CREDENTIALS_KEY" default:"Pjk+k4hske5KkKtbaKSVDOgpllRl+0EI6oCAdx88XqI="`
}

func GetConfig() Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}
