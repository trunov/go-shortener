package handler

import (
	"context"
	"crypto/aes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/trunov/go-shortener/internal/app/middleware"
	"github.com/trunov/go-shortener/internal/app/storage/postgres"
	"github.com/trunov/go-shortener/internal/app/util"

	"github.com/go-chi/chi/v5"
)

type Storager interface {
	Get(key string) (string, error)
	Add(key, link, userID string) string
	GetAllLinksByUserID(userID, baseURL string) []util.AllURLSResponse
	AddInBatch(br []util.BatchResponse, baseURL string)
}

type Handler struct {
	storage Storager
	pinger  postgres.Pinger
	baseURL string
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

func NewHandler(storage Storager, pinger postgres.Pinger, baseURL string) *Handler {
	return &Handler{storage: storage, pinger: pinger, baseURL: baseURL}
}

func (c *Handler) ShortenJSONLink(w http.ResponseWriter, r *http.Request) {
	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)

	key := util.GenerateRandomString()

	s := c.storage.Add(key, req.URL, userID)

	w.Header().Set("Content-Type", "application/json")

	if s != "" {
		finalRes := c.baseURL + "/" + s

		w.WriteHeader(http.StatusConflict)
		res := Response{Result: finalRes}
		if err := json.NewEncoder(w).Encode(res); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

	s := c.storage.Add(key, string(b), userID)

	w.Header().Set("Content-Type", "plain/text")

	if s != "" {
		finalRes := c.baseURL + "/" + s

		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(finalRes))
		return
	}

	w.WriteHeader(http.StatusCreated)

	finalRes := c.baseURL + "/" + key
	w.Write([]byte(finalRes))
}

func (c *Handler) GetURLLink(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	v, err := c.storage.Get(key)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Location", v)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (c *Handler) GetUrlsByUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	allURLSByUserID := c.storage.GetAllLinksByUserID(userID, c.baseURL)

	w.Header().Set("Content-Type", "application/json")

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

	c.storage.AddInBatch(batchRes, c.baseURL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(batchRes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
		r.Get("/user/urls", c.GetUrlsByUserID)

		r.Route("/shorten", func(r chi.Router) {
			r.Post("/", c.ShortenJSONLink)
			r.Post("/batch", c.ShortenLinksInBatch)
		})
	})

	return r
}
