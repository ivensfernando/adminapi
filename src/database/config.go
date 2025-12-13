package database

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	LogLevel  string `envconfig:"LOG_LEVEL" default:"debug"` // Expected to hold values like "debug", "info", "warn", "error"
	LogFormat string `envconfig:"LOG_FORMAT" default:"text"` // Expected to hold values like "json" or "text"
	EnableDB  bool   `envconfig:"ENABLE_DB" default:"false"`
	//DatabaseURLMain     string `envconfig:"DATABASE_URL_MAIN" default:"postgres://postgres:test123@localhost/postgres?sslmode=disable"`
	//DatabaseURLReadOnly string `envconfig:"DATABASE_URL_READONLY" default:"postgres://postgres:test123@localhost/postgres?sslmode=disable"`
	DatabaseURLMain     string `envconfig:"DATABASE_URL_MAIN" default:"postgres://u1fmo9gicp6qdm:P22fe8eaba2a32d0889d73980df41f10716a0c1974946218c3a30b4ccae1b54d4*123@51.158.130.226:22941/bot?sslmode=disable"`
	DatabaseURLReadOnly string `envconfig:"DATABASE_URL_READONLY" default:"postgres://u1fmo9gicp6qdm:P22fe8eaba2a32d0889d73980df41f10716a0c1974946218c3a30b4ccae1b54d4*123@51.158.130.226:22941/rdb?sslmode=disable"`
	GormLogLevel        int    `envconfig:"GORM_LOG_LEVEL" default:"2"`
}

func GetConfig() Config {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		panic(fmt.Errorf("error processing env config: %w", err))
	}
	return config
}
