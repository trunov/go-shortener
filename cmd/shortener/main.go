package main

import (
	_ "embed"
	"fmt"

	"github.com/trunov/go-shortener/internal/app/config"
	"github.com/trunov/go-shortener/internal/app/util"
)

//go:generate sh -c "printf %s $(git rev-parse HEAD) > commit.txt"
//go:embed commit.txt
var BuildCommit string

var (
	Version   string
	BuildDate string
)

func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	Version = util.DefaultIfEmpty(Version, "N/A")
	BuildDate = util.DefaultIfEmpty(BuildDate, "N/A")
	BuildCommit = util.DefaultIfEmpty(BuildCommit, "N/A")

	fmt.Printf("Build version: %s\n", Version)
	fmt.Printf("Build date: %s\n", BuildDate)
	fmt.Printf("Build commit: %s\n", BuildCommit)

	if err := StartServer(cfg); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
}
