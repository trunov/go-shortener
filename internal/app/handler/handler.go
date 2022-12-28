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
	"github.com/trunov/go-shortener/internal/app/middleware"
	"github.com/trunov/go-shortener/internal/app/storage/postgres"
	"github.com/trunov/go-shortener/internal/app/util"

	"github.com/go-chi/chi/v5"
)

type Storager interface {
	Get(ctx context.Context, key string) (string, error)
	Add(ctx context.Context, key, link, userID string) error
	GetAllLinksByUserID(ctx context.Context, userID, baseURL string) ([]util.AllURLSResponse, error)
	AddInBatch(ctx context.Context, br []util.BatchResponse, baseURL string) (string, error)
	GetShortenKey(ctx context.Context, originalURL string) (string, error)
	DeleteURLS(ctx context.Context, userID string, shortenURLS []string) error
}

type Worker interface {
	Start(ctx context.Context, inputCh chan []string, userID string)
}

type Handler struct {
	storage    Storager
	pinger     postgres.Pinger
	baseURL    string
	workerpool Worker
}

type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type Request struct {
	URL string `json:"URL"`
}

type Response struct {
	Result string `json:"result"`
}

func NewHandler(storage Storager, pinger postgres.Pinger, baseURL string, workerpool Worker) *Handler {
	return &Handler{storage: storage, pinger: pinger, baseURL: baseURL, workerpool: workerpool}
}

func (c *Handler) ShortenJSONLink(w http.ResponseWriter, r *http.Request) {
	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)

	key := util.GenerateRandomString()

	ctx := context.Background()
	err := c.storage.Add(ctx, key, req.URL, userID)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) || strings.Contains(err.Error(), "found entry") {
			k, err := c.storage.GetShortenKey(ctx, req.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			finalRes := c.baseURL + "/" + k

			w.WriteHeader(http.StatusConflict)
			res := Response{Result: finalRes}
			if err := json.NewEncoder(w).Encode(res); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	finalRes := c.baseURL + "/" + key

	res := Response{Result: finalRes}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Handler) ShortenLink(w http.ResponseWriter, r *http.Request) {
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

	ctx := context.Background()
	err = c.storage.Add(ctx, key, string(b), userID)

	w.Header().Set("Content-Type", "plain/text")

	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) || strings.Contains(err.Error(), "found entry") {
			k, err := c.storage.GetShortenKey(ctx, string(b))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			finalRes := c.baseURL + "/" + k

			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(finalRes))
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	finalRes := c.baseURL + "/" + key
	w.Write([]byte(finalRes))
}

func (c *Handler) GetURLLink(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	ctx := context.Background()
	v, err := c.storage.Get(ctx, key)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Location", v)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (c *Handler) GetUrlsByUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	ctx := context.Background()
	allURLSByUserID, err := c.storage.GetAllLinksByUserID(ctx, userID, c.baseURL)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(allURLSByUserID) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := json.NewEncoder(w).Encode(allURLSByUserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Handler) ShortenLinksInBatch(w http.ResponseWriter, r *http.Request) {
	var batchReq []BatchRequest

	if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)

	var batchRes []util.BatchResponse

	for _, v := range batchReq {
		key := util.GenerateRandomString()
		br := util.BatchResponse{CorrelationID: v.CorrelationID, ShortURL: c.baseURL + "/" + key, OriginalURL: v.OriginalURL, UserID: userID}
		batchRes = append(batchRes, br)
	}

	w.Header().Set("Content-Type", "application/json")

	ctx := context.Background()
	k, err := c.storage.AddInBatch(ctx, batchRes, c.baseURL)
	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) || strings.Contains(err.Error(), "found entry") {
			finalRes := c.baseURL + "/" + k

			w.WriteHeader(http.StatusConflict)

			res := Response{Result: finalRes}
			if err := json.NewEncoder(w).Encode(res); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(batchRes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := r.Context().Value("user_id").(string)

	var arr []string

	if err := json.NewDecoder(r.Body).Decode(&arr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	inputCh := util.GenerateChannel(arr)
	go c.workerpool.Start(ctx, inputCh, userID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}

func (c *Handler) PingDBHandler(w http.ResponseWriter, r *http.Request) {
	if c.pinger == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := c.pinger.Ping(context.Background()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		w.WriteHeader(http.StatusOK)
		return
	}
}

func NewRouter(c *Handler) chi.Router {
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
		r.Route("/user/urls", func(r chi.Router) {
			r.Get("/", c.GetUrlsByUserID)
			r.Delete("/", c.DeleteHandler)
		})

		r.Route("/shorten", func(r chi.Router) {
			r.Post("/", c.ShortenJSONLink)
			r.Post("/batch", c.ShortenLinksInBatch)
		})
	})

	return r
}
