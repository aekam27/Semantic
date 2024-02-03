package router

import (
	"goverse/pkg/elastic_service"
	"net/http"
)

type Route struct {
	Name        string
	Path        string
	Method      string
	HandlerFunc http.HandlerFunc
	Host        string
}

var routes = []Route{
	{
		"elasticGet",
		"/eget",
		"GET",
		elastic_service.GetProductsList,
		"*",
	},
}
