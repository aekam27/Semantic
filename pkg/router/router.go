package router

import "github.com/gorilla/mux"

func NewRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	// amw := authenticationMiddleware{tokenUsers: make(map[string]string)}
	// amw.Populate()
	// r.Use(amw.Middleware)
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
