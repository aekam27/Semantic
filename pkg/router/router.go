package router

import (
	"goverse/pkg/middleware"

	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	authmw := middleware.AuthTokenMap{Auth: make(map[string]interface{})}
	authmw.LoadTokens()
	r.Use(authmw.Middleware)
	r.Use(mux.CORSMethodMiddleware(r))
	sr := r.PathPrefix("/v1").Subrouter()
	for _, route := range routes {
		sr.HandleFunc(route.Path, route.HandlerFunc).Name(route.Name).Methods(route.Method)
		if route.Host != "*" {
			sr.Host(route.Host)
		}
	}
	return r
}
