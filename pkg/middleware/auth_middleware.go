package middleware

import (
	"fmt"
	"goverse/pkg/cache_service"
	conections "goverse/pkg/connections"
	util "goverse/pkg/utils"
	"net/http"
	"strings"
	"time"
)

type AuthTokenMap struct {
	Auth map[string]interface{}
}

var redisInitialized *cache_service.RedisCache

func gRedisService(auth *AuthTokenMap, stop chan bool, ticker *time.Ticker) error {
	if redisInitialized == nil {
		redisInitialized = cache_service.InitialzeRedis(conections.RedisCacheConn, conections.RedisContext)
	}
	tokens, err := redisInitialized.GetRecords("userToken")
	if err != nil {
		stop <- true
		defer ticker.Stop()
		return err
	}
	for _, value := range tokens["data"] {
		if tokenMap, ok := value.(map[string]interface{}); ok {
			token := tokenMap["token"].(string)
			expiryTime := tokenMap["exp"].(int64)
			currentTime := time.Now().Add(time.Minute).Unix()
			if expiryTime < currentTime {
				delete(auth.Auth, token)
				continue
			}
			auth.Auth[token] = tokenMap
		}
	}
	return nil
}

func (auth *AuthTokenMap) LoadTokens() {
	stop := make(chan bool)
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				gRedisService(auth, stop, ticker)
			case <-stop:
				fmt.Println("FIN Stopped")
				return
			}
		}
	}()
}

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
			mDetails := details.(map[string]interface{})
			currentTime := time.Now().Add(time.Minute).Unix()
			if mDetails["email"].(string) != claims["email"] {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if mDetails["exp"].(int64) != int64(claims["exp"].(float64)) || mDetails["exp"].(int64) < currentTime {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			// We found the token in our map
			fmt.Println("Authenticated user", "AKM", mDetails)
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}

	})
}
