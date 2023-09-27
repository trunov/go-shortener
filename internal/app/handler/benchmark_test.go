package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/trunov/go-shortener/internal/app/storage/memory"
	"github.com/trunov/go-shortener/internal/app/storage/postgres"
	"github.com/trunov/go-shortener/internal/app/util"
)

func BenchmarkShortenLink(b *testing.B) {
	keysLinksUserID := make(map[string]util.MapValue)
	s := memory.NewStorage(keysLinksUserID, "")
	var p postgres.Pinger
	baseURL := "http://localhost:8080"

	c := NewHandler(s, p, baseURL, nil)

	for i := 0; i < b.N; i++ {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://go.dev/src/net/http/request_test.go"))
		w := httptest.NewRecorder()
		h := http.HandlerFunc(c.ShortenLink)
		h.ServeHTTP(w, request)
	}
}

func BenchmarkGetLink(b *testing.B) {
	keysLinksUserID := map[string]util.MapValue{"12345678": {Link: "https://go.dev/src/net/http/request_test.go", UserID: "user1"}}
	s := memory.NewStorage(keysLinksUserID, "")
	var baseURL string
	var p postgres.Pinger

	c := NewHandler(s, p, baseURL, nil)
	r := NewRouter(c)

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/12345678", nil)
		newRecorder := httptest.NewRecorder()
		r.ServeHTTP(newRecorder, req)
	}
}
