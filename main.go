package main

import (
	"goverse/pkg/router"
	"log"
	"net/http"
)

func main() {
	r := router.NewRouter()
	log.Fatal(http.ListenAndServe(":8000", r))
}
