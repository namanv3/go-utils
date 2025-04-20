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
	return CorsHandlerMiddlewareForHeaders(nil, nil)(next)

}

func CorsHandlerMiddlewareForHeaders(allowedHeaders, requestedAllowOriginURL *string) func(next http.Handler) http.Handler {
	headers := "Content-Type, Authorization"
	if allowedHeaders != nil {
		headers = *allowedHeaders
	}
	allowOriginURL := "http://localhost"
	if requestedAllowOriginURL != nil {
		allowOriginURL = *requestedAllowOriginURL
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowOriginURL)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", headers)

			// Preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
