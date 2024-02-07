package router

import (
	"goverse/pkg/auth_service"
	"goverse/pkg/semantic_search_service"
	"net/http"
)

type Route struct {
	Name        string
	Path        string
	Method      string
	HandlerFunc http.HandlerFunc
	Host        string
	Auth        bool
}

var routes = []Route{
	{
		"mongoVectorSerach",
		"/mget",
		"POST",
		semantic_search_service.MongoVectorSearch,
		"*",
		true,
	},
	{
		"getToken",
		"/generateToken",
		"POST",
		auth_service.GenerateToken,
		"*",
		false,
	},
}
