// Package router wires handlers onto HTTP routes.
package router

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/aekam27/Semantic/pkg/auth_service"
	"github.com/aekam27/Semantic/pkg/middleware"
	"github.com/aekam27/Semantic/pkg/semantic_search_service"
)

// NewRouter builds the HTTP handler tree.
//
// Routes used to be declared in a table that carried a Host field, but every
// entry set it to "*" and the value was passed to mux.Route.Host, whose return
// value was discarded — so it never filtered anything. The table is gone and
// routes are declared directly.
func NewRouter(
	auth *middleware.Authenticator,
	authHandler *auth_service.Handler,
	searchHandler *semantic_search_service.Handler,
) *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	r.Use(middleware.Recover)
	r.Use(mux.CORSMethodMiddleware(r))

	r.HandleFunc("/healthz", health).Methods(http.MethodGet).Name("health")

	v1 := r.PathPrefix("/v1").Subrouter()

	// Public: exchanges an email for a token.
	v1.HandleFunc("/generateToken", authHandler.GenerateToken).
		Methods(http.MethodPost).
		Name("getToken")

	// Everything below requires a bearer token.
	secured := v1.NewRoute().Subrouter()
	secured.Use(auth.Middleware)
	secured.HandleFunc("/mget", searchHandler.MongoVectorSearch).
		Methods(http.MethodPost).
		Name("mongoVectorSearch")

	return r
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
