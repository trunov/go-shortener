// Package config allows you to read config from local calls or flags from the terminal.
package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

// Config represents the configuration with BaseURL, ServerAddress, FileStoragePath, and DatabaseDSN.
type Config struct {
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

// ReadConfig reads the configuration from environment variables and flags.
func ReadConfig() (Config, error) {
	cfgEnv := Config{}

	if err := env.Parse(&cfgEnv); err != nil {
		return cfgEnv, err
	}

	cfgFlag := Config{}

	flag.StringVar(&cfgFlag.BaseURL, "b", cfgEnv.BaseURL, "base URL")
	flag.StringVar(&cfgFlag.ServerAddress, "a", cfgEnv.ServerAddress, "server address")
	flag.StringVar(&cfgFlag.FileStoragePath, "f", cfgEnv.FileStoragePath, "file storage path")
	flag.StringVar(&cfgFlag.DatabaseDSN, "d", "", "database DSN")

	flag.Parse()

	return cfgFlag, nil
}
