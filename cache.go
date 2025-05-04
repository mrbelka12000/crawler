package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultCacheExpiration = 10 * time.Minute
	defaultCachevalue      = "taken"
)

type (
	Cache struct {
		redisClient *redis.Client
	}
)

func ConnectToRedis() (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &Cache{
		redisClient: rdb,
	}, nil
}

func (c *Cache) IsUsed(ctx context.Context, key string) bool {
	_, err := c.redisClient.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		c.Set(ctx, key)
		return false
	}

	return true
}

func (c *Cache) Set(ctx context.Context, key string) {
	c.redisClient.Set(ctx, key, defaultCachevalue, defaultCacheExpiration)
}
