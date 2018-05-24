package middleware

import (
	"fmt"
	"net/http"
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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", "application/json"))
		next.ServeHTTP(w, r)
	}
}
