package redisutil

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type RedisGetter interface {
	Get(context.Context, string) *redis.StringCmd
}

func JSONGet[T any](ctx context.Context, c RedisGetter, key string) (*T, error) {
	payload, err := c.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return UnmarshalGzipJSON[T](payload)
}

func UnmarshalGzipJSON[T any](payload string) (*T, error) {
	raw := []byte(payload)

	if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
		// Decode directly, if it does not start with the gzip magic bytes.
		var v T
		err := json.Unmarshal([]byte(payload), &v)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &v, nil
	}

	buf := bytes.NewBuffer(raw)
	zr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer zr.Close()

	var v T
	err = json.NewDecoder(zr).Decode(&v)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &v, nil
}

type RedisSetter interface {
	Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd
	GetSet(context.Context, string, interface{}) *redis.StringCmd
}

func GzipJSONSet[T any](ctx context.Context, c RedisSetter, key string, v T, expiration time.Duration) error {
	payload, err := MarshalGzipJSON(v)
	if err != nil {
		return err
	}

	err = c.Set(ctx, key, string(payload), expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

func GzipJSONGetSet[T any](ctx context.Context, c RedisSetter, key string, v T) (bool, error) {
	payload, err := MarshalGzipJSON(v)
	if err != nil {
		return false, err
	}

	old, err := c.GetSet(ctx, key, string(payload)).Result()
	if err == redis.Nil {
		// Return true even if the actual value is empty or nil, because
		// deleting a key would be its own funktion.
		return true, nil
	}
	if err != nil {
		return false, err
	}

	return old != string(payload), nil
}

func MarshalGzipJSON[T any](v T) (string, error) {
	jsonBuf, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	if len(jsonBuf) < 100 {
		return string(jsonBuf), nil
	}

	resultBuf := new(bytes.Buffer)
	resultWriter, err := gzip.NewWriterLevel(resultBuf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	_, err = resultWriter.Write(jsonBuf)
	if err != nil {
		return "", err
	}

	err = resultWriter.Close()
	if err != nil {
		return "", err
	}

	return resultBuf.String(), nil
}
