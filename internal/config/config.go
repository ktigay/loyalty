package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

const (
	defaultServerHost          = ":8081"
	defaultLogLevel            = "debug"
	defaultDatabaseDSN         = "postgres://postgres:postgres@192.168.1.46:5429/loyalty?sslmode=disable"
	defaultAuthSecret          = "auth_secret"
	defaultAccrualHost         = "http://localhost:8091/"
	defaultActualizeInterval   = 5
	defaultAccrualTimeout      = 1
	defaultAccrualMaxRateLimit = 3
)

// Config Конфигурация приложения.
type Config struct {
	ServerHost               string `env:"RUN_ADDRESS"`
	LogLevel                 string `env:"LOG_LEVEL"`
	DatabaseDSN              string `env:"DATABASE_URI"`
	AuthSecret               string `env:"SECRET"`
	AccrualHost              string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	AccrualActualizeInterval int    `env:"ACTUALIZE_INTERVAL"`
	AccrualTimeout           int    `env:"ACCRUAL_TIMEOUT"`
	AccrualMaxRateLimit      int    `env:"ACCRUAL_MAX_RATE_LIMIT"`
}

// New Создаёт конфигурацию.
func New(args []string) (*Config, error) {
	var err error

	cfg := Config{
		ServerHost:               defaultServerHost,
		LogLevel:                 defaultLogLevel,
		DatabaseDSN:              defaultDatabaseDSN,
		AuthSecret:               defaultAuthSecret,
		AccrualHost:              defaultAccrualHost,
		AccrualActualizeInterval: defaultActualizeInterval,
		AccrualTimeout:           defaultAccrualTimeout,
		AccrualMaxRateLimit:      defaultAccrualMaxRateLimit,
	}
	if err = env.Parse(&cfg); err != nil {
		return nil, err
	}

	// приоритет у флагов.
	flags := flag.NewFlagSet("server flags", flag.ContinueOnError)
	flags.StringVar(&cfg.ServerHost, "a", cfg.ServerHost, "address and port to run server")
	flags.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "log level")
	flags.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database DSN")
	flags.StringVar(&cfg.AuthSecret, "s", cfg.AuthSecret, "Authorization secret key")
	flags.StringVar(&cfg.AccrualHost, "r", cfg.AccrualHost, "Accrual host")

	if err = flags.Parse(args); err != nil {
		return nil, err
	}

	if cfg.ServerHost == "" {
		return nil, fmt.Errorf("host flag is required")
	}

	return &cfg, nil
}
