package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type RedisRepository interface {
	Set(string, interface{}) error
	Get(string, interface{}) (bool, error)
	DeleteAll() error
	Close() error
}

type redisRepository struct {
	rdb    *redis.Client
	prefix string
}

func NewRedisRepository(addr, prefix string) (RedisRepository, error) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("can't connect redis error while ping %w", err)
	}

	return &redisRepository{
		rdb: rdb,
	}, nil
}

func (r *redisRepository) Set(k string, v interface{}) error {
	ctx := context.Background()
	o, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return r.rdb.Set(ctx, r.prefix+k, string(o), 0).Err()
}

func (r *redisRepository) Get(k string, v interface{}) (bool, error) {
	ctx := context.Background()
	val, err := r.rdb.Get(ctx, r.prefix+k).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}

	if err == redis.Nil || val == "" {
		return false, nil
	}

	return true, json.Unmarshal([]byte(val), v)
}

func (r *redisRepository) DeleteAll() error {
	ctx := context.Background()
	iter := r.rdb.Scan(ctx, 0, r.prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		if err := r.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}

func (r *redisRepository) Close() error {
	return r.rdb.Close()
}
