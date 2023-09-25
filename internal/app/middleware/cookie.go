// Package middleware provides middleware functions for http requests.
package middleware

import (
	"context"
	"net/http"

	"github.com/trunov/go-shortener/internal/app/encryption"
	"github.com/trunov/go-shortener/internal/app/util"
)

// cookieName represents the name of the cookie that will store the user ID.
const cookieName = "user_id"

// ctxName is the key under which user ID will be stored in the context.
var ctxName interface{} = "user_id"

// CookieMiddleware is a middleware that ensures that each request has a user ID associated with it.
// If an incoming request has a "user_id" cookie, it decodes its value and adds it to the request's context.
// If the cookie is missing or cannot be decoded, a new user ID is generated, encoded, and set as a cookie
// before adding it to the request's context. The middleware uses the provided encryption key for encoding
// and decoding user IDs.
//
// The middleware relies on the encryption and util packages to handle the encoding/decoding
// and user ID generation respectively.
//
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(middleware.CookieMiddleware(yourEncryptionKey))
//	...
//
//	// Inside your handler:
//	userID := r.Context().Value("user_id").(string)
func CookieMiddleware(key []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookieUserID, _ := r.Cookie(cookieName)
			encryptor := encryption.NewEncryptor(key)

			if cookieUserID != nil {
				userID, err := encryptor.Decode(cookieUserID.Value)

				if err == nil {
					ctx := context.WithValue(r.Context(), ctxName, userID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			userID, err := util.GenerateRandomUserID()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			encoded, err := encryptor.Encode([]byte(userID))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			cookie := &http.Cookie{Name: "user_id", Value: encoded}
			http.SetCookie(w, cookie)

			ctx := context.WithValue(r.Context(), ctxName, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
