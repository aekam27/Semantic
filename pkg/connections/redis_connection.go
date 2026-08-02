package connections

import (
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"

	"github.com/aekam27/Semantic/pkg/config"
)

// NewRedisCache builds the token cache client.
//
// go-redis dials lazily, so this cannot fail here; a bad address surfaces on
// the first cache operation.
func NewRedisCache(cfg config.RedisConfig) *cache.Cache {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server1": cfg.Addr,
		},
	})

	return cache.New(&cache.Options{
		Redis:      ring,
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})
}
