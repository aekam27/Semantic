package util

import (
	"bytes"
	"encoding/json"
	"goverse/pkg/cache_service"
	conections "goverse/pkg/connections"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var redisInitialized *cache_service.RedisCache

func gsRedisService(email, token string, exp int64) error {
	if redisInitialized == nil {
		redisInitialized = cache_service.InitialzeRedis(conections.RedisCacheConn, conections.RedisContext)
	}
	tokens, err := redisInitialized.GetRecords("userToken")
	if err != nil {
		if err.Error() == "cache: key is missing" {
			tokens = make(map[string][]interface{})
			tokens["data"] = []interface{}{}
		} else {
			return err
		}
	}
	putObj := &map[string]interface{}{"email": email, "hotlisted": false, "token": token, "exp": exp}
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
	exp := time.Now().Add(time.Minute * 15).Unix()
	claims["exp"] = exp
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := at.SignedString([]byte("tokensecret"))
	if err != nil {
		return "", err
	}
	_ = gsRedisService(email, token, exp)
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

func PostApi(url string, doc interface{}) ([]byte, error) {
	method := "POST"
	requestByte, err := json.Marshal(doc)
	if err != nil {
		return []byte{}, err
	}
	requestReader := bytes.NewReader(requestByte)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, requestReader)
	if err != nil {
		return []byte{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}
