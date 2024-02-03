package cache_service

import (
	"context"
	"time"

	"github.com/go-redis/cache/v9"
)

type RedisCache struct {
	ctx   context.Context
	cache *cache.Cache
}

func (rc *RedisCache) GetContext() context.Context {
	return rc.ctx
}

func (rc *RedisCache) GetCacheRef() *cache.Cache {
	return rc.cache
}

func (rc *RedisCache) GetRecords(key string) (map[string][]interface{}, error) {
	var (
		res map[string][]interface{}
		err error
	)

	if err = rc.cache.Get(rc.ctx, key, &res); err == nil {
		return res, nil
	}
	return nil, err
}

func (rc *RedisCache) PutRecords(key string, val interface{}) error {
	if err := rc.cache.Set(&cache.Item{
		Ctx:   rc.ctx,
		Key:   key,
		Value: val,
		TTL:   time.Hour,
	}); err != nil {
		return err
	}
	return nil
}

func InitialzeRedis(cache *cache.Cache, ctx context.Context) *RedisCache {
	return &RedisCache{
		cache: cache,
		ctx:   ctx,
	}
}
