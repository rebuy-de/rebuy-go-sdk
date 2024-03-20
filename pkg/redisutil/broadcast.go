package redisutil

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	broadcastValueField = "data"
)

type BroadcastRediser interface {
	XAdd(ctx context.Context, a *redis.XAddArgs) *redis.StringCmd
	XRead(ctx context.Context, a *redis.XReadArgs) *redis.XStreamSliceCmd
}

type Broadcast[T any] struct {
	client BroadcastRediser
	key    string
}

func NewBroadcast[T any](client BroadcastRediser, key string) (*Broadcast[T], error) {
	return &Broadcast[T]{
		client: client,
		key:    key,
	}, nil
}

func (b *Broadcast[T]) Add(ctx context.Context, value *T) error {
	payload, err := MarshalGzipJSON(value)
	if err != nil {
		return errors.WithStack(err)
	}

	args := &redis.XAddArgs{
		Stream: b.key,
		MaxLen: 10,
		Approx: true,
		Values: map[string]interface{}{
			broadcastValueField: payload,
		},
	}

	err = b.client.XAdd(ctx, args).Err()
	return errors.WithStack(err)
}

func (b *Broadcast[T]) Read(ctx context.Context, id string) (*T, string, error) {
	args := &redis.XReadArgs{
		Streams: []string{b.key, id},
		Count:   1,
		Block:   time.Minute,
	}

	streams, err := b.client.XRead(ctx, args).Result()
	if err != nil {
		return nil, id, errors.WithStack(err)
	}

	for _, stream := range streams {
		for _, sm := range stream.Messages {
			payload := sm.Values[broadcastValueField].(string)
			value, err := UnmarshalGzipJSON[T](payload)
			if err != nil {
				return nil, id, errors.WithStack(err)
			}

			//lint:ignore SA4004 We just want to have the first message and
			//returning withing two loops is easier than checking lengths.
			return value, sm.ID, nil
		}
	}

	return nil, id, errors.Errorf("no data")
}
