package main

import (
	"context"
	"crypto/aes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/trunov/go-shortener/proto"

	"github.com/trunov/go-shortener/internal/app/config"
	"github.com/trunov/go-shortener/internal/app/file"
	"github.com/trunov/go-shortener/internal/app/grpcService"
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

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen on gRPC port: %v", err)
	}

	key, err := util.GenerateRandom(2 * aes.BlockSize)
	if err != nil {
		return err
	}

	gs := grpc.NewServer(grpc.UnaryInterceptor(grpcService.AuthInterceptor(key)))
	newGrpc := grpcService.NewGrpcServer(c)

	pb.RegisterUrlShortenerServer(gs, &newGrpc)

	reflection.Register(gs)

	go func() {
		log.Printf("gRPC server is listening on port %s", cfg.GRPCPort)
		if err := gs.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	<-done
	log.Print("Shutdown signal received")

	// Stop accepting new requests.
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctxShutdown); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	gs.GracefulStop()
	log.Print("gRPC server stopped")

	// Finish processing ongoing work and stop the worker pool.
	workerpool.Stop()

	// Close database connections.
	if dbpool != nil {
		dbpool.Close()
	}

	log.Print("Server and Workerpool Gracefully Stopped")

	return nil
}
