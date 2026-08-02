// Package cache_service stores issued tokens so they can be revoked and so
// running instances share a view of what is currently valid.
package cache_service

import (
	"context"
	"time"

	"github.com/go-redis/cache/v9"
)

// tokenSetKey is the cache key holding every live token.
const tokenSetKey = "userToken"

// TokenRecord is one issued token.
//
// This used to be an untyped map[string]interface{} that callers picked apart
// with unchecked type assertions. Those panicked whenever the msgpack round
// trip through Redis returned a different numeric type than the writer used.
type TokenRecord struct {
	Email     string `msgpack:"email"`
	Token     string `msgpack:"token"`
	Hotlisted bool   `msgpack:"hotlisted"`
	ExpiresAt int64  `msgpack:"exp"`
}

// Expired reports whether the record is past its expiry at time now.
func (t TokenRecord) Expired(now time.Time) bool {
	return t.ExpiresAt <= now.Unix()
}

// tokenSet is the stored shape: a struct rather than a bare slice so the
// payload can gain fields without breaking decoding.
type tokenSet struct {
	Data []TokenRecord `msgpack:"data"`
}

// TokenStore reads and writes the live token set.
type TokenStore struct {
	cache *cache.Cache
	ttl   time.Duration
}

// NewTokenStore wraps a cache client. The ttl bounds how long the whole token
// set survives without a write.
func NewTokenStore(c *cache.Cache, ttl time.Duration) *TokenStore {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &TokenStore{cache: c, ttl: ttl}
}

// List returns every stored token. A missing key is not an error: it just means
// no token has been issued yet.
func (s *TokenStore) List(ctx context.Context) ([]TokenRecord, error) {
	var set tokenSet
	err := s.cache.Get(ctx, tokenSetKey, &set)
	if err == cache.ErrCacheMiss {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return set.Data, nil
}

// Add stores a token, dropping any that have already expired so the set does
// not grow without bound.
//
// This is a read-modify-write against a single key, so two instances issuing
// tokens at the same instant can lose one of them. Tokens are re-issuable, so
// the cost is a client retry rather than data loss.
func (s *TokenStore) Add(ctx context.Context, record TokenRecord) error {
	existing, err := s.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	kept := make([]TokenRecord, 0, len(existing)+1)
	for _, t := range existing {
		if !t.Expired(now) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, record)

	return s.cache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   tokenSetKey,
		Value: tokenSet{Data: kept},
		TTL:   s.ttl,
	})
}
