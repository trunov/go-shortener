package postgres

import (
	"context"

	"github.com/jackc/pgx"
	"github.com/trunov/go-shortener/internal/app/util"
)

type Pinger interface {
	Ping(context.Context) error
}

type dbStorage struct {
	conn *pgx.Conn
}

func NewDBStorage(conn *pgx.Conn) *dbStorage {
	return &dbStorage{conn: conn}
}

func (s *dbStorage) Get(ctx context.Context, key string) (string, error) {
	var v string

	err := s.conn.QueryRowEx(ctx, "SELECT original_url from shortener WHERE short_url = $1", nil, key).Scan(&v)
	if err != nil {
		return "", err
	}

	return v, nil
}

func (s *dbStorage) GetShortenKey(ctx context.Context, originalURL string) (string, error) {
	var v string

	err := s.conn.QueryRowEx(ctx, "SELECT short_url from shortener WHERE original_url = $1", nil, originalURL).Scan(&v)
	if err != nil {
		return "", err
	}

	return v, nil
}

func (s *dbStorage) Add(ctx context.Context, key, link, userID string) error {
	_, err := s.conn.ExecEx(ctx, "INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", nil, key, link, userID)

	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) GetAllLinksByUserID(ctx context.Context, userID, baseURL string) ([]util.AllURLSResponse, error) {
	allUrls := []util.AllURLSResponse{}

	rows, err := s.conn.QueryEx(ctx, "SELECT short_url, original_url, user_id from shortener", nil)

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

func (s *dbStorage) AddInBatch(ctx context.Context, br []util.BatchResponse, baseURL string) (string, error) {
	tx, err := s.conn.Begin()
	if err != nil {
		return "", err
	}

	defer tx.Rollback()

	for _, v := range br {
		if _, err := tx.ExecEx(ctx, "INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", nil, v.ShortURL[len(baseURL)+1:], v.OriginalURL, v.UserID); err != nil {
			return v.ShortURL, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return "", nil
}

func (s *dbStorage) Ping(ctx context.Context) error {
	err := s.conn.Ping(ctx)

	if err != nil {
		return err
	}
	return nil
}
