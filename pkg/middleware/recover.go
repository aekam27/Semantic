package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recover turns a panic in a handler into a 500 instead of tearing down the
// connection and logging a bare stack trace from net/http.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic serving %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				http.Error(w, "Server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
