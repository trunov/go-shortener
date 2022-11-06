package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/trunov/go-shortener/internal/app/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ShortenLink(t *testing.T) {
	type want struct {
		code        int
		key         string
		contentType string
	}

	tests := []struct {
		name         string
		want         want
		linksAndKeys map[string]string
		website      string
	}{
		{
			name: "should return 201 and generated key",
			want: want{
				code:        201,
				key:         "12345678",
				contentType: "plain/text",
			},
			linksAndKeys: make(map[string]string),
			website:      "https://go.dev/src/net/http/request_test.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keysAndLinks := make(map[string]string)
			s := storage.NewStorage(keysAndLinks, "")
			var baseURL string

			c := NewContainer(s, baseURL)
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.website))

			w := httptest.NewRecorder()
			h := http.HandlerFunc(c.ShortenLink)

			h.ServeHTTP(w, request)
			res := w.Result()

			defer res.Body.Close()

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)

			assert.Equal(t, len(string(b)), 8)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func Test_GetLink(t *testing.T) {
	type want struct {
		statusCode int
		url        string
	}

	tests := []struct {
		name         string
		want         want
		keysAndLinks map[string]string
		key          string
	}{
		{
			name: "should return url in header location and status of temporary redirection",
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				url:        "https://go.dev/src/net/http/request_test.go",
			},
			keysAndLinks: map[string]string{"12345678": "https://go.dev/src/net/http/request_test.go"},
			key:          "12345678",
		},
		{
			name: "should return 404 Not Found",
			want: want{
				statusCode: http.StatusNotFound,
				url:        "",
			},
			keysAndLinks: map[string]string{"123asd1": "https://go.dev/src/net/http/request_test.go"},
			key:          "21323",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewStorage(tt.keysAndLinks, "")
			var baseURL string

			c := NewContainer(s, baseURL)

			r := NewRouter(c)

			req, err := http.NewRequest(http.MethodGet, "/"+tt.key, nil)
			require.NoError(t, err)

			newRecorder := httptest.NewRecorder()
			r.ServeHTTP(newRecorder, req)

			res := newRecorder.Result()

			if res.StatusCode == http.StatusTemporaryRedirect {
				assert.Equal(t, res.Header.Get("Location"), tt.want.url)
			}

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
		})
	}

}
