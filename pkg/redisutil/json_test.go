package redisutil

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testJSONData struct {
	ID    int
	Stuff string
}

func TestGzipJSONSetGet(t *testing.T) {
	fake := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: fake.Addr(),
	})

	ctx := context.Background()
	long := testJSONData{
		ID:    42,
		Stuff: strings.Repeat("ha", 100),
	}

	t.Run("Set", func(t *testing.T) {
		err := GzipJSONSet(ctx, client, "test", long, time.Hour)
		require.NoError(t, err)

		payload, err := fake.Get("test")
		require.NoError(t, err)
		require.Equal(t, []byte{0x1f, 0x8b}, []byte(payload)[0:2])
	})

	t.Run("Get", func(t *testing.T) {
		retrieved, err := JSONGet[testJSONData](ctx, client, "test")
		require.NoError(t, err)
		require.Equal(t, &long, retrieved)
	})
}

func TestGzipJSONSetGetChanged(t *testing.T) {
	fake := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: fake.Addr(),
	})

	ctx := context.Background()
	long := testJSONData{
		ID:    42,
		Stuff: strings.Repeat("ha", 100),
	}

	t.Run("SetInitial", func(t *testing.T) {
		changed, err := GzipJSONGetSet(ctx, client, "test", long)
		require.NoError(t, err)
		assert.True(t, changed)

		payload, err := fake.Get("test")
		require.NoError(t, err)
		require.Equal(t, []byte{0x1f, 0x8b}, []byte(payload)[0:2])
	})

	t.Run("SetSame", func(t *testing.T) {
		changed, err := GzipJSONGetSet(ctx, client, "test", long)
		require.NoError(t, err)
		assert.False(t, changed)

		payload, err := fake.Get("test")
		require.NoError(t, err)
		require.Equal(t, []byte{0x1f, 0x8b}, []byte(payload)[0:2])
	})

	t.Run("Get", func(t *testing.T) {
		retrieved, err := JSONGet[testJSONData](ctx, client, "test")
		require.NoError(t, err)
		require.Equal(t, &long, retrieved)
	})
}
