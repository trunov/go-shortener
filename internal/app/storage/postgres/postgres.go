package postgres

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx"
	"github.com/trunov/go-shortener/internal/app/util"
)

type Pinger interface {
	Ping(context.Context) error
}

type dbStorage struct {
	conn *pgx.Conn
}

func NewDbStorage(conn *pgx.Conn) *dbStorage {
	return &dbStorage{conn: conn}
}

func (s *dbStorage) Get(key string) (string, error) {
	var v string
	err := s.conn.QueryRow("SELECT original_url from shortener WHERE short_url = $1", key).Scan(&v)
	if err != nil {
		return "", err
	}

	return v, nil
}

func (s *dbStorage) Add(key, link, userID string) string {
	_, err := s.conn.Exec("INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", key, link, userID)

	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
			var v string
			s.conn.QueryRow("SELECT short_url from shortener WHERE original_url = $1", link).Scan(&v)

			return v
		}

		log.Fatal(err)
	}

	return ""
}

func (s *dbStorage) GetAllLinksByUserID(userID, baseURL string) []util.AllURLSResponse {
	allUrls := []util.AllURLSResponse{}

	rows, err := s.conn.Query("SELECT short_url, original_url, user_id from shortener")

	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {
		var shortURL, originalURL, dbUserID string
		err = rows.Scan(&shortURL, &originalURL, &dbUserID)
		if err != nil {
			log.Fatal(err)
		}

		if userID == dbUserID {
			allUrls = append(allUrls, util.AllURLSResponse{ShortURL: baseURL + "/" + shortURL, OriginalURL: originalURL})
		}
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return allUrls
}

func (s *dbStorage) AddInBatch(br []util.BatchResponse, baseURL string) {
	tx, err := s.conn.Begin()
	if err != nil {
		log.Fatal(err)
	}

	defer tx.Rollback()

	for _, v := range br {
		if _, err := tx.Exec("INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", v.ShortURL[len(baseURL)+1:], v.OriginalURL, v.UserID); err != nil {
			log.Fatal(err)
		}
	}

	tx.Commit()
}

func (s *dbStorage) Ping(ctx context.Context) error {
	err := s.conn.Ping(ctx)

	if err != nil {
		return err
	}
	return nil
}
