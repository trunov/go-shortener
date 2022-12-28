package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/trunov/go-shortener/internal/app/config"
	"github.com/trunov/go-shortener/internal/app/file"
	"github.com/trunov/go-shortener/internal/app/handler"
	"github.com/trunov/go-shortener/internal/app/storage/memory"
	"github.com/trunov/go-shortener/internal/app/storage/postgres"
	"github.com/trunov/go-shortener/internal/app/util"
	"github.com/trunov/go-shortener/migrate"
)

func StartServer(cfg config.Config) {
	keysAndLinks := make(map[string]util.MapValue)
	ctx := context.Background()

	if cfg.FileStoragePath != "" {
		reader, err := file.SeedMapWithKeysAndLinks(cfg.FileStoragePath, keysAndLinks)
		if err != nil {
			log.Fatal(err)
		}
		defer reader.Close()
	}

	var storage handler.Storager
	var pinger postgres.Pinger

	var dbpool *pgxpool.Pool
	if cfg.DatabaseDSN != "" {
		var err error
		dbpool, err = pgxpool.Connect(ctx, cfg.DatabaseDSN)
		if err != nil {
			fmt.Printf("Unable to connect to database: %v\n", err)
			os.Exit(1)
		}
		defer dbpool.Close()

		dbStorage := postgres.NewDBStorage(dbpool)
		storage = dbStorage
		pinger = dbStorage

		err = migrate.Migrate(cfg.DatabaseDSN, migrate.Migrations)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		storage = memory.NewStorage(keysAndLinks, cfg.FileStoragePath)
	}
	workerpool := NewWorkerpool(&storage)

	c := handler.NewHandler(storage, pinger, cfg.BaseURL, workerpool)
	r := handler.NewRouter(c)

	log.Println("server is starting on port ", cfg.ServerAddress)
	http.ListenAndServe(cfg.ServerAddress, r)
}
