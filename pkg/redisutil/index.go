package redisutil

import (
	"context"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type RedisIndexer interface {
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	MGet(ctx context.Context, keys ...string) *redis.SliceCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
}

func IndexVacuum(ctx context.Context, c RedisIndexer, indexKey string, dataKeyPrefix Prefix) error {
	ids, err := c.SMembers(ctx, indexKey).Result()
	if err != nil {
		return errors.Wrap(err, "failed to get ids")
	}

	if len(ids) == 0 {
		return nil
	}

	keys := dataKeyPrefix.Keys(ids)

	values, err := c.MGet(ctx, keys...).Result()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve values")
	}

	expired := []interface{}{}
	for k, value := range values {
		id := ids[k]
		if value == nil {
			expired = append(expired, id)
		}
	}

	if len(expired) == 0 {
		return nil
	}

	err = c.SRem(ctx, indexKey, expired...).Err()
	return errors.Wrap(err, "failed to delete expired keys")
}
