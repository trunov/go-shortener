package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipWriter wraps the http.ResponseWriter and an io.Writer. It's used to write compressed
// responses to the client.
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Write writes the provided byte slice to the gzipWriter's underlying io.Writer.
func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// GzipHandle is a middleware that compresses the HTTP response using GZIP compression
// if the client's "Accept-Encoding" header includes "gzip". The compressed response
// will include a "Content-Encoding: gzip" header.
//
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(middleware.GzipHandle)
//	...
func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

// DecompressHandle is a middleware that decompresses incoming HTTP requests with a
// "Content-Encoding: gzip" header. After decompression, the request's body will be
// replaced with the decompressed content, allowing downstream handlers to read
// the uncompressed request body.
//
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(middleware.DecompressHandle)
//	...
func DecompressHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}

		defer gz.Close()

		r.Body = gz
		next.ServeHTTP(w, r)
	})
}
