package database

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	LogLevel            string `envconfig:"LOG_LEVEL" default:"debug"` // Expected to hold values like "debug", "info", "warn", "error"
	LogFormat           string `envconfig:"LOG_FORMAT" default:"text"` // Expected to hold values like "json" or "text"
	EnableDB            bool   `envconfig:"ENABLE_DB" default:"false"`
	DatabaseURLMain     string `envconfig:"DATABASE_URL_MAIN" default:"postgres://postgres:test123@localhost/postgres?sslmode=disable"`
	DatabaseURLReadOnly string `envconfig:"DATABASE_URL_READONLY" default:"postgres://postgres:test123@localhost/postgres?sslmode=disable"`
	GormLogLevel        int    `envconfig:"GORM_LOG_LEVEL" default:"2"`
}

func GetConfig() Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}
