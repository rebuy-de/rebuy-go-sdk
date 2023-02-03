package redisutil

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexVacuum(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	var (
		indexKey   = "index"
		dataPrefix = Prefix("data")
	)

	t.Run("NoIndex", func(t *testing.T) {
		mr.FlushAll()
		err := IndexVacuum(context.Background(), redisClient, indexKey, dataPrefix)
		assert.NoError(t, err)
	})

	t.Run("EmptyIndex", func(t *testing.T) {
		mr.FlushAll()
		mr.SAdd(indexKey)
		err := IndexVacuum(context.Background(), redisClient, indexKey, dataPrefix)
		assert.NoError(t, err)
	})

	t.Run("NoExpiredIndex", func(t *testing.T) {
		mr.FlushAll()
		mr.SAdd(indexKey, "foobar")
		mr.Set(dataPrefix.Key("foobar"), "something")
		err := IndexVacuum(context.Background(), redisClient, indexKey, dataPrefix)
		assert.NoError(t, err)
	})

	t.Run("SimpleExpire", func(t *testing.T) {
		mr.FlushAll()
		mr.SAdd(indexKey, "foo", "bar")
		mr.Set(dataPrefix.Key("bar"), "blubber")
		err := IndexVacuum(context.Background(), redisClient, indexKey, dataPrefix)
		assert.NoError(t, err)

		ids, err := mr.Members(indexKey)
		assert.NoError(t, err)
		assert.Equal(t, []string{"bar"}, ids)
	})
}
