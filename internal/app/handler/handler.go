// Package handler provides HTTP handlers to manage and interact with URL shortening functionality.
package handler

import (
	"context"
	"crypto/aes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/jackc/pgerrcode"

	"github.com/trunov/go-shortener/internal/app/middleware"
	"github.com/trunov/go-shortener/internal/app/storage/postgres"
	"github.com/trunov/go-shortener/internal/app/util"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// Storager is an interface that outlines the required operations
// to store, retrieve and manage shortened URLs.
type Storager interface {
	Get(ctx context.Context, key string) (util.ShortenerGet, error)
	Add(ctx context.Context, key, link, userID string) error
	GetAllLinksByUserID(ctx context.Context, userID, baseURL string) ([]util.AllURLSResponse, error)
	AddInBatch(ctx context.Context, br []util.BatchResponse, baseURL string) (string, error)
	GetShortenKey(ctx context.Context, originalURL string) (string, error)
	DeleteURLS(ctx context.Context, userID string, shortenURLS []string) error
	GetInternalStats(ctx context.Context) (util.InternalStats, error)
}

// Worker is an interface for starting background tasks.
type Worker interface {
	Start(ctx context.Context, inputCh chan []string, userID string)
}

// Handler contains all the dependencies to handle HTTP requests for
// the URL shortening application.
type Handler struct {
	storage       Storager
	pinger        postgres.Pinger
	baseURL       string
	trustedSubnet string
	workerpool    Worker
}

// BatchRequest represents a single URL shortening request in a batch operation.
type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// Request represents a request to shorten a given URL.
type Request struct {
	URL string `json:"URL"`
}

// Response provides a shortened URL in response to a shortening request.
type Response struct {
	Result string `json:"result"`
}

// NewHandler initializes a new Handler with the provided dependencies.
func NewHandler(storage Storager, pinger postgres.Pinger, baseURL, trustedSubnet string, workerpool Worker) *Handler {
	return &Handler{storage: storage, pinger: pinger, baseURL: baseURL, trustedSubnet: trustedSubnet, workerpool: workerpool}
}

// ShortenJSONLink handles the request to shorten a link provided as JSON.
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

// core_logic
// I think it would be correct to just return error
func (c *Handler) ProcessShortenLink(url, userID string) (string, int, error) {
	key := util.GenerateRandomString()
	ctx := context.Background()
	err := c.storage.Add(ctx, key, url, userID)

	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) || strings.Contains(err.Error(), "found entry") {
			k, err := c.storage.GetShortenKey(ctx, url)
			if err != nil {
				return "", http.StatusInternalServerError, err
			}
			finalRes := c.baseURL + "/" + k
			return finalRes, http.StatusConflict, nil
		}
		return "", http.StatusInternalServerError, err
	}

	finalRes := c.baseURL + "/" + key
	return finalRes, http.StatusCreated, nil
}

// ShortenLink handles the request to shorten a link provided as plain text.
func (c *Handler) ShortenLink(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "user ID not found", http.StatusBadRequest)
		return
	}

	finalRes, statusCode, err := c.ProcessShortenLink(string(b), userID)

	w.Header().Set("Content-Type", "plain/text")
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	w.WriteHeader(statusCode)
	w.Write([]byte(finalRes))
}

func (c *Handler) RetrieveURL(ctx context.Context, key string) (string, bool, error) {
	v, err := c.storage.Get(ctx, key)
	if err != nil {
		return "", false, err
	}

	if v.IsDeleted {
		return "", true, nil
	}

	return v.OriginalURL, false, nil
}

// GetURLLink redirects the user to the original URL using the shortened key.
func (c *Handler) GetURLLink(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	ctx := context.Background()

	originalURL, isDeleted, err := c.RetrieveURL(ctx, key)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if isDeleted {
		w.WriteHeader(http.StatusGone)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (c *Handler) RetrieveURLsByUserID(ctx context.Context, userID string) ([]util.AllURLSResponse, error) {
	allURLsByUserID, err := c.storage.GetAllLinksByUserID(ctx, userID, c.baseURL)
	if err != nil {
		return nil, err
	}

	return allURLsByUserID, nil
}

// GetUrlsByUserID retrieves all the URLs shortened by a particular user.
func (c *Handler) GetUrlsByUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	ctx := context.Background()

	allURLsByUserID, err := c.RetrieveURLsByUserID(ctx, userID)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(allURLsByUserID) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := json.NewEncoder(w).Encode(allURLsByUserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Handler) ProcessBatchShortening(ctx context.Context, batchReq []BatchRequest, userID string) ([]util.BatchResponse, string, error) {
	var batchRes []util.BatchResponse

	for _, v := range batchReq {
		key := util.GenerateRandomString()
		br := util.BatchResponse{CorrelationID: v.CorrelationID, ShortURL: c.baseURL + "/" + key, OriginalURL: v.OriginalURL, UserID: userID}
		batchRes = append(batchRes, br)
	}

	k, err := c.storage.AddInBatch(ctx, batchRes, c.baseURL)
	if err != nil {
		return batchRes, k, err
	}

	return batchRes, "", nil
}

// ShortenLinksInBatch handles batch requests to shorten multiple links.
func (c *Handler) ShortenLinksInBatch(w http.ResponseWriter, r *http.Request) {
	var batchReq []BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)
	ctx := context.Background()

	batchRes, k, err := c.ProcessBatchShortening(ctx, batchReq, userID)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) || strings.Contains(err.Error(), "found entry") {
			finalRes := c.baseURL + "/" + k
			w.WriteHeader(http.StatusConflict)
			res := Response{Result: finalRes}
			if err := json.NewEncoder(w).Encode(res); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
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

func (c *Handler) ProcessDeletion(ctx context.Context, urls []string, userID string) error {
	inputCh := util.GenerateChannel(urls)
	go c.workerpool.Start(ctx, inputCh, userID)
	return nil
}

// DeleteHandler handles the request to delete specific shortened links.
func (c *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := r.Context().Value("user_id").(string)

	var arr []string

	if err := json.NewDecoder(r.Body).Decode(&arr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.ProcessDeletion(ctx, arr, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}

// PingDBHandler checks the health of the connected database.
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

func (c *Handler) CheckClientIP(ipStr string) bool {
	if c.trustedSubnet == "" || ipStr == "" {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	_, ipNet, err := net.ParseCIDR(c.trustedSubnet)
	if err != nil {
		return false
	}

	return ipNet.Contains(ip)
}

func (c *Handler) RetrieveInternalStats(ctx context.Context) (util.InternalStats, error) {
	return c.storage.GetInternalStats(ctx)
}

// GetInternalStats return Internal stats of shortener service such as amount of total Urls and Users.
func (c *Handler) GetInternalStats(w http.ResponseWriter, r *http.Request) {
	ipStr := r.Header.Get("X-Real-IP")

	if !c.CheckClientIP(ipStr) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	ctx := context.Background()
	stats, err := c.RetrieveInternalStats(ctx)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// NewRouter sets up and returns a new router with all the URL shortening routes configured.
func NewRouter(c *Handler) (chi.Router, error) {
	r := chi.NewRouter()

	key, err := util.GenerateRandom(2 * aes.BlockSize)
	if err != nil {
		return nil, err
	}

	r.Use(middleware.GzipHandle)
	r.Use(middleware.DecompressHandle)
	r.Use(middleware.CookieMiddleware(key))
	r.Mount("/debug", chiMiddleware.Profiler())

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

		r.Route("/internal", func(r chi.Router) {
			r.Get("/stats", c.GetInternalStats)
		})
	})

	return r, nil
}
