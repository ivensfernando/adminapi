package executor

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
}

func GetConfig() *Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return &config
}
