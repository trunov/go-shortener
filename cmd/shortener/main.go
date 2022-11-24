package main

import (
	"log"
	"net/http"

	"github.com/trunov/go-shortener/internal/app/config"
	"github.com/trunov/go-shortener/internal/app/file"
	"github.com/trunov/go-shortener/internal/app/handler"
	"github.com/trunov/go-shortener/internal/app/storage"
)

func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	keysAndLinks := make(map[string]string)

	if cfg.FileStoragePath != "" {
		reader := file.SeedMapWithKeysAndLinks(cfg.FileStoragePath, keysAndLinks)
		defer reader.Close()
	}

	s := storage.NewStorage(keysAndLinks, cfg.FileStoragePath)

	c := handler.NewContainer(s, cfg.BaseURL)

	r := handler.NewRouter(c)

	log.Println("server is starting on port ", cfg.ServerAddress)
	http.ListenAndServe(cfg.ServerAddress, r)
}
