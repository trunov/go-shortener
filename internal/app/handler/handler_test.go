package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/trunov/go-shortener/internal/app/storage/memory"
	"github.com/trunov/go-shortener/internal/app/storage/postgres"
	"github.com/trunov/go-shortener/internal/app/util"

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
		name    string
		want    want
		website string
	}{
		{
			name: "should return 201 and generated key",
			want: want{
				code:        201,
				key:         "12345678",
				contentType: "plain/text",
			},
			website: "https://go.dev/src/net/http/request_test.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keysLinksUserID := make(map[string]util.MapValue)
			s := memory.NewStorage(keysLinksUserID, "")
			var p postgres.Pinger
			baseURL := "http://localhost:8080"

			c := NewHandler(s, p, baseURL)
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.website))

			w := httptest.NewRecorder()
			h := http.HandlerFunc(c.ShortenLink)

			h.ServeHTTP(w, request)
			res := w.Result()

			defer res.Body.Close()

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)

			assert.Equal(t, true, strings.Contains(string(b), "http://localhost:8080"))
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
		name            string
		want            want
		keysLinksUserID map[string]util.MapValue
		key             string
	}{
		{
			name: "should return url in header location and status of temporary redirection",
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				url:        "https://go.dev/src/net/http/request_test.go",
			},
			keysLinksUserID: map[string]util.MapValue{"12345678": {Link: "https://go.dev/src/net/http/request_test.go", UserID: "user1"}},
			key:             "12345678",
		},
		{
			name: "should return 404 Not Found",
			want: want{
				statusCode: http.StatusNotFound,
				url:        "",
			},
			keysLinksUserID: map[string]util.MapValue{"123asd1": {Link: "https://go.dev/src/net/http/request_test.go", UserID: "user1"}},
			key:             "21323",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := memory.NewStorage(tt.keysLinksUserID, "")

			var baseURL string
			var p postgres.Pinger

			c := NewHandler(s, p, baseURL)

			r := NewRouter(c)

			req, err := http.NewRequest(http.MethodGet, "/"+tt.key, nil)
			require.NoError(t, err)

			newRecorder := httptest.NewRecorder()
			r.ServeHTTP(newRecorder, req)

			res := newRecorder.Result()
			defer res.Body.Close()

			if res.StatusCode == http.StatusTemporaryRedirect {
				assert.Equal(t, res.Header.Get("Location"), tt.want.url)
			}

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
		})
	}

}
