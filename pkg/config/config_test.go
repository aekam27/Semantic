package config

import (
	"testing"
	"time"
)

const validSecret = "a-test-secret-that-is-long-enough-32"

func TestLoadRequiresJWTSecret(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when JWT_SECRET is unset")
	}
}

func TestLoadRejectsShortJWTSecret(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	t.Setenv("JWT_SECRET", "too-short")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a short JWT_SECRET")
	}
}

func TestLoadRequiresMongoURI(t *testing.T) {
	t.Setenv("JWT_SECRET", validSecret)

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when MONGODB_URI is unset")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Addr != ":8001" {
		t.Errorf("Addr = %q, want :8001", cfg.Addr)
	}
	if cfg.TokenTTL != 15*time.Minute {
		t.Errorf("TokenTTL = %v, want 15m", cfg.TokenTTL)
	}
	if cfg.Mongo.Collection != "vectormap" {
		t.Errorf("Collection = %q, want vectormap", cfg.Mongo.Collection)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %q, want localhost:6379", cfg.Redis.Addr)
	}
}

func TestLoadReadsOverrides(t *testing.T) {
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("MONGODB_URI", "mongodb://example:27017")
	t.Setenv("ADDR", ":9000")
	t.Setenv("MONGODB_DATABASE", "other")
	t.Setenv("MONGODB_COLLECTION", "vectors")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("EMBEDDING_SERVICE_URL", "http://embed:8000/v1/vectorEmbeddings/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Addr != ":9000" {
		t.Errorf("Addr = %q, want :9000", cfg.Addr)
	}
	if cfg.Mongo.Database != "other" || cfg.Mongo.Collection != "vectors" {
		t.Errorf("Mongo = %+v, want database=other collection=vectors", cfg.Mongo)
	}
	if cfg.EmbeddingServiceURL != "http://embed:8000/v1/vectorEmbeddings/" {
		t.Errorf("EmbeddingServiceURL = %q", cfg.EmbeddingServiceURL)
	}
}

func TestDurationEnvAcceptsBothForms(t *testing.T) {
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")

	t.Setenv("TOKEN_TTL", "45m")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TokenTTL != 45*time.Minute {
		t.Errorf("TokenTTL = %v, want 45m", cfg.TokenTTL)
	}

	t.Setenv("TOKEN_TTL", "90")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TokenTTL != 90*time.Second {
		t.Errorf("TokenTTL = %v, want 90s", cfg.TokenTTL)
	}

	t.Setenv("TOKEN_TTL", "not-a-duration")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for an unparseable TOKEN_TTL")
	}
}
