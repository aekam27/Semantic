package router

import (
	"goverse/pkg/middleware"

	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	authmw := middleware.AuthTokenMap{Auth: make(map[string]interface{})}
	authmw.LoadTokens()
	r.Use(mux.CORSMethodMiddleware(r))
	for _, route := range routes {
		sr := r.PathPrefix("/v1").Subrouter()
		sr.HandleFunc(route.Path, route.HandlerFunc).Name(route.Name).Methods(route.Method)
		if route.Host != "*" {
			sr.Host(route.Host)
		}
		if route.Auth {
			sr.Use(authmw.Middleware)
		}
	}
	return r
}
