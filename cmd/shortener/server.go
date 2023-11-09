package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/crypto/acme/autocert"

	"github.com/trunov/go-shortener/internal/app/config"
	"github.com/trunov/go-shortener/internal/app/file"
	"github.com/trunov/go-shortener/internal/app/handler"
	"github.com/trunov/go-shortener/internal/app/storage/memory"
	"github.com/trunov/go-shortener/internal/app/storage/postgres"
	"github.com/trunov/go-shortener/internal/app/util"
	"github.com/trunov/go-shortener/migrate"
)

func StartServer(cfg config.Config) error {
	var server *http.Server
	keysAndLinks := make(map[string]util.MapValue)
	ctx := context.Background()

	if cfg.FileStoragePath != "" {
		reader, err := file.SeedMapWithKeysAndLinks(cfg.FileStoragePath, keysAndLinks)
		if err != nil {
			return err
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
			return err
		}
		defer dbpool.Close()

		dbStorage := postgres.NewDBStorage(dbpool)
		storage = dbStorage
		pinger = dbStorage

		err = migrate.Migrate(cfg.DatabaseDSN, migrate.Migrations)
		if err != nil {
			return err
		}
	} else {
		storage = memory.NewStorage(keysAndLinks, cfg.FileStoragePath)
	}
	workerpool := NewWorkerpool(&storage)

	c := handler.NewHandler(storage, pinger, cfg.BaseURL, cfg.TrustedSubnet, workerpool)
	r, err := handler.NewRouter(c)
	if err != nil {
		fmt.Printf("Failed to create router: %v\n", err)
		return err
	}

	if cfg.EnableHTTPS {
		manager := &autocert.Manager{
			Cache:      autocert.DirCache("cache-dir"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("myshortener.ru", "www.myshortener.ru"),
		}

		server = &http.Server{
			Addr:      cfg.ServerAddress,
			Handler:   r,
			TLSConfig: manager.TLSConfig(),
		}

	} else {
		server = &http.Server{
			Addr:    cfg.ServerAddress,
			Handler: r,
		}
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() error {
		if err := server.ListenAndServe(); err != nil {
			return err
		}
		return nil
	}()

	log.Println("server is starting on port ", cfg.ServerAddress)

	<-done
	log.Print("Shutdown signal received")

	// Stop accepting new requests.
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctxShutdown); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	// Finish processing ongoing work and stop the worker pool.
	workerpool.Stop()

	// Close database connections.
	if dbpool != nil {
		dbpool.Close()
	}

	log.Print("Server and Workerpool Gracefully Stopped")

	return nil
}
