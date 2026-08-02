// Package config loads runtime settings from the environment.
//
// Nothing here is hardcoded: credentials and endpoints previously lived as
// empty string literals in the connection packages, which meant the server
// could not start without editing source.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds every setting the server needs to run.
type Config struct {
	Addr                string
	JWTSecret           []byte
	TokenTTL            time.Duration
	Mongo               MongoConfig
	Redis               RedisConfig
	EmbeddingServiceURL string
	RequestTimeout      time.Duration
}

// MongoConfig describes the vector search database.
type MongoConfig struct {
	URI        string
	Database   string
	Collection string
}

// RedisConfig describes the token cache.
type RedisConfig struct {
	Addr string
}

// Load reads configuration from the environment, applying defaults for
// everything that has a safe one. It returns an error rather than exiting so
// callers decide how to handle a bad environment.
func Load() (Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET must be set")
	}
	if len(secret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 bytes, got %d", len(secret))
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		return Config{}, fmt.Errorf("MONGODB_URI must be set")
	}

	tokenTTL, err := durationEnv("TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	requestTimeout, err := durationEnv("REQUEST_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:      stringEnv("ADDR", ":8001"),
		JWTSecret: []byte(secret),
		TokenTTL:  tokenTTL,
		Mongo: MongoConfig{
			URI:        mongoURI,
			Database:   stringEnv("MONGODB_DATABASE", "semantic"),
			Collection: stringEnv("MONGODB_COLLECTION", "vectormap"),
		},
		Redis: RedisConfig{
			Addr: stringEnv("REDIS_ADDR", "localhost:6379"),
		},
		EmbeddingServiceURL: stringEnv("EMBEDDING_SERVICE_URL", "http://localhost:8000/v1/vectorEmbeddings/"),
		RequestTimeout:      requestTimeout,
	}, nil
}

func stringEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	// Accept both "15m" and a bare number of seconds.
	if d, err := time.ParseDuration(raw); err == nil {
		return d, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 15m or a number of seconds: %w", key, err)
	}
	return time.Duration(seconds) * time.Second, nil
}
