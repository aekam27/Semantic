package middleware

import (
	"fmt"
	"goverse/pkg/cache_service"
	conections "goverse/pkg/connections"
	util "goverse/pkg/utils"
	"log"
	"net/http"
	"strings"
)

type AuthTokenMap struct {
	Auth map[string]interface{}
}

var redisInitialized *cache_service.RedisCache

func gRedisService(auth *AuthTokenMap) error {
	if redisInitialized == nil {
		redisInitialized = cache_service.InitialzeRedis(conections.RedisCacheConn, conections.RedisContext)
	}
	tokens, err := redisInitialized.GetRecords("userToken")
	if err != nil {
		return err
	}
	for _, entry := range tokens["data"] {
		// Access and print the data
		for key, value := range entry {
			fmt.Printf("%s: %v\n", key, value)
		}
	}
	for _, value := range tokens["data"] {
		token := value["token"]
		auth.Auth[token] = value
	}
	return nil
}

func (auth *AuthTokenMap) loadToken() {
	gRedisService(auth)
}

// Middleware function, which will be called for each request
func (auth *AuthTokenMap) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := strings.Split(r.Header.Get("Authorization"), " ")

		//Check for valid token
		if len(tokenString) < 2 {
			http.Error(w, "Forbidden", http.StatusUnauthorized)
			return
		}

		//Check if token exist in inmemorymap
		if details, found := auth.Auth[tokenString[1]]; found {
			claims, err := util.DecodeToken(tokenString[1])
			if err != nil {
				http.Error(w, "Forbidden", http.StatusUnauthorized)
				return
			}
			// We found the token in our map
			log.Printf("Authenticated user %s\n", user)
			// Pass down the request to the next middleware (or final handler)
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}

	})
}
