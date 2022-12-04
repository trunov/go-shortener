package main

import (
	"log"

	"github.com/trunov/go-shortener/internal/app/config"
)

func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	StartServer(cfg)
}
