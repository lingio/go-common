package middleware

import (
	"fmt"
	"net/http"
	"github.com/rs/cors"
)

func AllowOriginHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	}
}

func ContentTypeHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", "application/json"))
		next.ServeHTTP(w, r)
	}
}

func StandardHeadersHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", "application/json"))
		next.ServeHTTP(w, r)
	}
}

func CORSControlHandler(next http.Handler) http.Handler {
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	return corsMiddleware.Handler(next)
}
