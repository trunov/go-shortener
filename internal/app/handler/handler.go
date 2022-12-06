package handler

import (
	"context"
	"crypto/aes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx"
	"github.com/trunov/go-shortener/internal/app/middleware"
	"github.com/trunov/go-shortener/internal/app/storage"
	"github.com/trunov/go-shortener/internal/app/util"

	"github.com/go-chi/chi/v5"
)

type Container struct {
	conn    *pgx.Conn
	storage storage.Storager
	baseURL string
}

type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type Request struct {
	URL string `json:"URL"`
}

type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
	originalURL   string
	userID        string
}

type Response struct {
	Result string `json:"result"`
}

func NewContainer(conn *pgx.Conn, storage storage.Storager, baseURL string) *Container {
	return &Container{conn: conn, storage: storage, baseURL: baseURL}
}

func (c *Container) ShortenJSONLink(w http.ResponseWriter, r *http.Request) {
	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)

	key := util.GenerateRandomString()

	if c.conn != nil {
		_, err := c.conn.Exec("INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", key, req.URL, userID)

		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
			var v string

			c.conn.QueryRow("SELECT short_url from shortener WHERE original_url = $1", req.URL).Scan(&v)

			finalRes := c.baseURL + "/" + v

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			res := Response{Result: finalRes}
			if err := json.NewEncoder(w).Encode(res); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		if err != nil {
			log.Fatal(err)
		}
	} else {
		c.storage.Add(key, req.URL, userID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	finalRes := c.baseURL + "/" + key

	res := Response{Result: finalRes}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Container) ShortenLink(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	userID, ok := r.Context().Value("user_id").(string)

	// without ok check first test was failing
	if !ok {
		fmt.Println("")
	}

	key := util.GenerateRandomString()

	if c.conn != nil {
		_, err := c.conn.Exec("INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", key, string(b), userID)

		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
			var v string

			c.conn.QueryRow("SELECT short_url from shortener WHERE original_url = $1", string(b)).Scan(&v)

			finalRes := c.baseURL + "/" + v

			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(finalRes))
			return
		}

		if err != nil {
			log.Fatal(err)
		}
	} else {
		c.storage.Add(key, string(b), userID)
	}

	w.Header().Set("Content-Type", "plain/text")
	w.WriteHeader(http.StatusCreated)

	finalRes := c.baseURL + "/" + key
	w.Write([]byte(finalRes))
}

func (c *Container) GetURLLink(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	var v string
	if c.conn != nil {
		err := c.conn.QueryRow("SELECT original_url from shortener WHERE short_url = $1", key).Scan(&v)
		if err == pgx.ErrNoRows {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	} else {
		var err error
		v, err = c.storage.Get(key)

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Location", v)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (c *Container) GetUrlsByUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	allURLSByUserID := util.FindAllURLSByUserID(c.storage.GetAll(), userID, c.baseURL)

	if len(allURLSByUserID) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(allURLSByUserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Container) ShortenLinksInBatch(w http.ResponseWriter, r *http.Request) {
	var batchReq []BatchRequest

	if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)

	var batchRes []BatchResponse

	for _, v := range batchReq {
		key := util.GenerateRandomString()

		br := BatchResponse{CorrelationID: v.CorrelationID, ShortURL: c.baseURL + "/" + key, originalURL: v.OriginalURL, userID: userID}

		batchRes = append(batchRes, br)
	}

	if c.conn != nil {
		tx, err := c.conn.Begin()
		if err != nil {
			log.Fatal(err)
		}

		defer tx.Rollback()

		for _, v := range batchRes {
			if _, err := tx.Exec("INSERT INTO shortener (short_url, original_url, user_id) values ($1, $2,$3)", v.ShortURL[len(c.baseURL)+1:], v.originalURL, v.userID); err != nil {
				log.Fatal(err)
			}
		}

		tx.Commit()
	} else {
		for _, v := range batchRes {
			c.storage.Add(v.ShortURL[len(c.baseURL)+1:], v.originalURL, v.userID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(batchRes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Container) PingDBHandler(w http.ResponseWriter, r *http.Request) {
	if c.conn != nil {
		err := c.conn.Ping(context.Background())

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func NewRouter(c *Container) chi.Router {
	r := chi.NewRouter()

	key, err := util.GenerateRandom(2 * aes.BlockSize)
	if err != nil {
		log.Fatal(err)
	}

	r.Use(middleware.GzipHandle)
	r.Use(middleware.DecompressHandle)
	r.Use(middleware.CookieMiddleware(key))

	r.Post("/", c.ShortenLink)
	r.Get("/{key}", c.GetURLLink)
	r.Get("/ping", c.PingDBHandler)

	r.Route("/api", func(r chi.Router) {
		r.Get("/user/urls", c.GetUrlsByUserID)

		r.Route("/shorten", func(r chi.Router) {
			r.Post("/", c.ShortenJSONLink)
			r.Post("/batch", c.ShortenLinksInBatch)
		})
	})

	return r
}
