package helpers

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

type APINameKey struct{}
type UserKey struct{}

func HandlerNameMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		if route == nil {
			next.ServeHTTP(w, r)
			return
		}
		apiName := route.GetName()
		ctx := context.WithValue(r.Context(), APINameKey{}, apiName)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CorsHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
