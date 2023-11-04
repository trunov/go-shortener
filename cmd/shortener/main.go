package main

import (
	_ "embed"
	"fmt"

	"github.com/trunov/go-shortener/internal/app/config"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	if err := StartServer(cfg); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
}
