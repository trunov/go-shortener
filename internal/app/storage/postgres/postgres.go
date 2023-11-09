// Package postgres provides a PostgreSQL-backed storage implementation for the URL shortener.
package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/trunov/go-shortener/internal/app/util"
)

// Pinger represents an interface that can verify if a connection is alive.
type Pinger interface {
	Ping(context.Context) error
}

// dbStorage is a database storage implementation using a PostgreSQL connection pool.
type dbStorage struct {
	dbpool *pgxpool.Pool
}

// NewDBStorage creates a new instance of dbStorage with a given connection pool.
func NewDBStorage(conn *pgxpool.Pool) *dbStorage {
	return &dbStorage{dbpool: conn}
}

// Get retrieves the original URL and its deletion status associated with a given key from the database.
func (s *dbStorage) Get(ctx context.Context, key string) (util.ShortenerGet, error) {
	var shortener util.ShortenerGet

	err := s.dbpool.QueryRow(ctx, "SELECT original_url, is_deleted from shortener WHERE short_url = $1", key).Scan(&shortener.OriginalURL, &shortener.IsDeleted)
	if err != nil {
		return shortener, err
	}

	return shortener, nil
}

// GetShortenKey finds and returns the key for a given original URL in the database.
func (s *dbStorage) GetShortenKey(ctx context.Context, originalURL string) (string, error) {
	var v string

	err := s.dbpool.QueryRow(ctx, "SELECT short_url from shortener WHERE original_url = $1", originalURL).Scan(&v)
	if err != nil {
		return "", err
	}

	return v, nil
}

// Add inserts a new shortened URL entry into the database.
func (s *dbStorage) Add(ctx context.Context, key, link, userID string) error {
	_, err := s.dbpool.Exec(ctx, "INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", key, link, userID)

	if err != nil {
		return err
	}

	return nil
}

// GetAllLinksByUserID fetches all the short URLs associated with a user ID from the database and returns them.
func (s *dbStorage) GetAllLinksByUserID(ctx context.Context, userID, baseURL string) ([]util.AllURLSResponse, error) {
	allUrls := []util.AllURLSResponse{}

	rows, err := s.dbpool.Query(ctx, "SELECT short_url, original_url, user_id from shortener")

	if err != nil {
		return allUrls, err
	}

	defer rows.Close()

	for rows.Next() {
		var shortURL, originalURL, dbUserID string
		err = rows.Scan(&shortURL, &originalURL, &dbUserID)
		if err != nil {
			return allUrls, err
		}

		if userID == dbUserID {
			allUrls = append(allUrls, util.AllURLSResponse{ShortURL: baseURL + "/" + shortURL, OriginalURL: originalURL})
		}
	}

	err = rows.Err()
	if err != nil {
		return allUrls, err
	}

	return allUrls, nil
}

// AddInBatch adds multiple shortened URLs at once to the database using a transaction.
func (s *dbStorage) AddInBatch(ctx context.Context, br []util.BatchResponse, baseURL string) (string, error) {
	tx, err := s.dbpool.Begin(ctx)
	if err != nil {
		return "", err
	}

	defer tx.Rollback(ctx)

	for _, v := range br {
		if _, err := tx.Exec(ctx, "INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", v.ShortURL[len(baseURL)+1:], v.OriginalURL, v.UserID); err != nil {
			return v.ShortURL, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return "", err
	}

	return "", nil
}

// DeleteURLS marks specified URLs as deleted for a given user ID in the database.
func (s *dbStorage) DeleteURLS(ctx context.Context, userID string, shortenURLS []string) error {
	tx, err := s.dbpool.Begin(ctx)
	if err != nil {
		return err
	}

	b := &pgx.Batch{}

	for _, shortenURL := range shortenURLS {
		sqlStatement := `
		UPDATE shortener
		SET is_deleted = $1
		WHERE short_url = $2
		AND user_id = $3;`

		b.Queue(sqlStatement, true, shortenURL, userID)
	}

	batchResults := tx.SendBatch(ctx, b)

	var qerr error
	var rows pgx.Rows
	for qerr == nil {
		rows, qerr = batchResults.Query()
		rows.Close()
	}

	if err = tx.Commit(ctx); err != nil {
		fmt.Println("error occurred in here")
		return err
	}

	log.Printf("I did set status of is_deleted to true to followed keys: %s\n", shortenURLS)

	return nil
}

// Ping checks the database connection status.
func (s *dbStorage) Ping(ctx context.Context) error {
	err := s.dbpool.Ping(ctx)

	if err != nil {
		return err
	}
	return nil
}

// GetInternalStats return Internal stats of shortener service such as Urls and Users.
func (s *dbStorage) GetInternalStats(ctx context.Context) (util.InternalStats, error) {
	stats := util.InternalStats{}

	query := `
		SELECT 
		  COUNT(DISTINCT user_id) AS total_users, 
		  COUNT(*) AS total_short_urls 
		FROM 
		  shortener
		WHERE 
		  is_deleted = false;
	`

	err := s.dbpool.QueryRow(ctx, query).Scan(&stats.Users, &stats.Urls)
	if err != nil {
		return stats, err
	}

	return stats, nil
}
