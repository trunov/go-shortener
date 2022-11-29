package handler

import (
	"encoding/json"
	"io"
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

	key := util.GenerateRandomString()

	c.storage.Add(key, req.URL)

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

	key := util.GenerateRandomString()

	c.storage.Add(key, string(b))

	w.Header().Set("Content-Type", "plain/text")
	w.WriteHeader(http.StatusCreated)

	finalRes := "http://localhost:8080/" + key
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

func NewRouter(c *Container) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.GzipHandle)
	r.Use(middleware.DecompressHandle)

	r.Post("/", c.ShortenLink)
	r.Post("/api/shorten", c.ShortenJSONLink)
	r.Get("/{key}", c.GetURLLink)

	return r
}
