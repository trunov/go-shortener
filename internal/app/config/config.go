// Package config allows you to read config from local calls, flags from the terminal or json config file.
package config

import (
	"errors"
	"flag"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultBaseURL         = "http://localhost:8080"
	defaultServerAddress   = "localhost:8080"
	defaultFileStoragePath = ""
	defaultDatabaseDSN     = ""
	defaultConfig          = ""
	defaultEnableHTTPS     = false
	defaultTrustedSubnet   = ""
)

func init() {
	viper.SetDefault("base_url", defaultBaseURL)
	viper.SetDefault("server_address", defaultServerAddress)
	viper.SetDefault("file_storage_path", defaultFileStoragePath)
	viper.SetDefault("database_dsn", defaultDatabaseDSN)
	viper.SetDefault("config", defaultConfig)
	viper.SetDefault("enable_https", defaultEnableHTTPS)
	viper.SetDefault("trusted_subnet", defaultTrustedSubnet)
}

// Config represents the configuration with BaseURL, ServerAddress, FileStoragePath, DatabaseDSN and EnableHTTPS
type Config struct {
	BaseURL         string
	ServerAddress   string
	FileStoragePath string
	DatabaseDSN     string
	EnableHTTPS     bool
	TrustedSubnet   string
}

func bindToFlag() {
	pflag.StringP("base_url", "b", defaultBaseURL, "base URL")
	pflag.StringP("server_address", "a", defaultServerAddress, "server address")
	pflag.StringP("file_storage_path", "f", defaultFileStoragePath, "file storage path")
	pflag.StringP("database_dsn", "d", defaultDatabaseDSN, "database DSN")
	pflag.StringP("config", "c", defaultConfig, "config file path")
	pflag.BoolP("enable_https", "s", defaultEnableHTTPS, "enable HTTPS")
	pflag.StringP("trusted_subnet", "t", defaultTrustedSubnet, "trusted subnet")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)
}

func readJSONConfigFile() error {
	configPath := viper.GetString("config")
	// ignore if empty
	if configPath == "" {
		return nil
	}

	fmt.Println(configPath)

	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return errors.New("config file not found")
		}
		return err
	}
	return nil
}

func bindToEnv() {
	viper.BindEnv("base_url", "BASE_URL")
	viper.BindEnv("server_address", "SERVER_ADDRESS")
	viper.BindEnv("file_storage_path", "FILE_STORAGE_PATH")
	viper.BindEnv("database_dsn", "DATABASE_DSN")
	viper.BindEnv("config", "CONFIG")
	viper.BindEnv("enable_https", "ENABLE_HTTPS")
	viper.BindEnv("trusted_subnet", "TRUSTED_SUBNET")
}

// ReadConfig reads the configuration from environment variables, flags and json config file.
func ReadConfig() (Config, error) {
	bindToFlag()
	bindToEnv()
	err := readJSONConfigFile()
	if err != nil {
		return Config{}, err
	}

	res := Config{
		BaseURL:         viper.GetString("base_url"),
		ServerAddress:   viper.GetString("server_address"),
		FileStoragePath: viper.GetString("file_storage_path"),
		DatabaseDSN:     viper.GetString("database_dsn"),
		EnableHTTPS:     viper.GetBool("enable_https"),
		TrustedSubnet:   viper.GetString("trusted_subnet"),
	}

	return res, nil
}
