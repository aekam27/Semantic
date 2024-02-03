package util

import (
	"goverse/pkg/cache_service"
	conections "goverse/pkg/connections"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var redisInitialized *cache_service.RedisCache

func gsRedisService(email, token string) error {
	if redisInitialized == nil {
		redisInitialized = cache_service.InitialzeRedis(conections.RedisCacheConn, conections.RedisContext)
	}
	tokens, err := redisInitialized.GetRecords("userToken")
	if err != nil {
		return err
	}
	putObj := &map[string]interface{}{"email": email, "hotlisted": false, "token": token}
	tokens["data"] = append(tokens["data"], *putObj)
	err = redisInitialized.PutRecords("userToken", tokens)
	if err != nil {
		return err
	}
	return nil
}

func GenerateToken(userid, email, name, status string) (string, error) {
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["email"] = email
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := at.SignedString([]byte("tokensecret"))
	if err != nil {
		return "", err
	}
	_ = gsRedisService(email, token)
	return token, nil
}

func DecodeToken(tok string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tok, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("tokensecret"), nil
	})
	if err != nil {
		return claims, err
	}
	return claims, nil
}
