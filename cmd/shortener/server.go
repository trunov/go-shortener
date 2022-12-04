package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx"
	"github.com/trunov/go-shortener/internal/app/config"
	"github.com/trunov/go-shortener/internal/app/file"
	"github.com/trunov/go-shortener/internal/app/handler"
	"github.com/trunov/go-shortener/internal/app/storage"
	"github.com/trunov/go-shortener/internal/app/util"
)

func StartServer(cfg config.Config) {
	keysAndLinks := make(map[string]util.MapValue)

	if cfg.FileStoragePath != "" {
		reader := file.SeedMapWithKeysAndLinks(cfg.FileStoragePath, keysAndLinks)
		defer reader.Close()
	}

	dbConfig, err := pgx.ParseConnectionString(cfg.DatabaseDSN)
	if err != nil {
		log.Println(err)
	}

	var conn *pgx.Conn
	if cfg.DatabaseDSN != "" {
		var err error
		conn, err = pgx.Connect(dbConfig)
		if err != nil {
			fmt.Printf("Unable to connect to database: %v\n", err)
			os.Exit(1)
		}
		defer conn.Close()
	}

	s := storage.NewStorage(keysAndLinks, cfg.FileStoragePath)
	c := handler.NewContainer(conn, s, cfg.BaseURL)
	r := handler.NewRouter(c)

	log.Println("server is starting on port ", cfg.ServerAddress)
	http.ListenAndServe(cfg.ServerAddress, r)
}