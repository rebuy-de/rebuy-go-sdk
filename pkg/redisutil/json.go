package redisutil

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisGetter interface {
	Get(context.Context, string) *redis.StringCmd
}

func JSONGet(ctx context.Context, c RedisGetter, key string, v interface{}) error {
	payload, err := c.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(payload), v)
}

type RedisSetter interface {
	Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd
}

func JSONSet(ctx context.Context, c RedisSetter, key string, v interface{}, expiration time.Duration) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return c.Set(ctx, key, string(payload), expiration).Err()
}
