package handler

import (
	"crypto/aes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/trunov/go-shortener/internal/app/middleware"
	"github.com/trunov/go-shortener/internal/app/storage"
	"github.com/trunov/go-shortener/internal/app/util"

	"github.com/go-chi/chi/v5"
)

type Container struct {
	storage storage.Storager
	baseURL string
}

type Request struct {
	URL string `json:"URL"`
}

type Response struct {
	Result string `json:"result"`
}

func NewContainer(storage storage.Storager, baseURL string) *Container {
	return &Container{storage: storage, baseURL: baseURL}
}

func (c *Container) ShortenJSONLink(w http.ResponseWriter, r *http.Request) {
	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)
	fmt.Println(userID)

	key := util.GenerateRandomString()

	c.storage.Add(key, req.URL, userID)

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
	userID, ok := r.Context().Value("user_id").(string)

	// without ok check first test was failing
	if !ok {
		fmt.Println("")
	}

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	key := util.GenerateRandomString()

	c.storage.Add(key, string(b), userID)

	w.Header().Set("Content-Type", "plain/text")
	w.WriteHeader(http.StatusCreated)

	finalRes := c.baseURL + "/" + key
	w.Write([]byte(finalRes))
}

func (c *Container) GetURLLink(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	v, err := c.storage.Get(key)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
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
	r.Post("/api/shorten", c.ShortenJSONLink)
	r.Get("/{key}", c.GetURLLink)
	r.Get("/api/user/urls", c.GetUrlsByUserID)

	return r
}
