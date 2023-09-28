package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/trunov/go-shortener/internal/app/storage/memory"
	"github.com/trunov/go-shortener/internal/app/storage/postgres"
	"github.com/trunov/go-shortener/internal/app/util"
)

func ExampleHandler_GetURLLink() {
	keysLinksUserID := map[string]util.MapValue{"123asd1": {Link: "https://www.example.com", UserID: "user1"}}
	s := memory.NewStorage(keysLinksUserID, "")
	baseURL := ""
	var p postgres.Pinger

	h := NewHandler(s, p, baseURL, nil)
	r := NewRouter(h)

	// Make a request to the shortened URL
	req, err := http.NewRequest(http.MethodGet, "/123asd1", nil)
	if err != nil {
		log.Fatal(err)
	}

	newRecorder := httptest.NewRecorder()
	r.ServeHTTP(newRecorder, req)

	// Check the result
	res := newRecorder.Result()
	defer res.Body.Close()

	if res.StatusCode == http.StatusTemporaryRedirect {
		fmt.Println("Redirect URL:", res.Header.Get("Location"))
	}
	fmt.Println("Status Code:", res.StatusCode)

	// Output:
	// Redirect URL: https://www.example.com
	// Status Code: 307
}

func ExampleHandler_ShortenLink() {
	const website = "https://go.dev/src/net/http/request_test.go"
	keysLinksUserID := make(map[string]util.MapValue)
	s := memory.NewStorage(keysLinksUserID, "")
	var p postgres.Pinger
	baseURL := "http://localhost:8080"

	c := NewHandler(s, p, baseURL, nil)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(website))

	w := httptest.NewRecorder()
	h := http.HandlerFunc(c.ShortenLink)

	h.ServeHTTP(w, request)
	res := w.Result()

	defer res.Body.Close()

	if res.StatusCode == http.StatusCreated {
		fmt.Printf("Status Code: %d", res.StatusCode)
	}

	// Output:
	// Status Code: 201
}
