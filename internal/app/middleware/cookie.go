package middleware

import (
	"context"
	"net/http"

	"github.com/trunov/go-shortener/internal/app/encryption"
	"github.com/trunov/go-shortener/internal/app/util"
)

const cookieName = "user_id"

var ctxName interface{} = "user_id"

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
