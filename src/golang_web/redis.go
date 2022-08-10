package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hashicorp/go-hclog"
)

type Redis struct {
	client *redis.Client
	logger hclog.Logger
}

func NewRedisClient(client *redis.Client, logger hclog.Logger) *Redis {
	return &Redis{
		client: client,
		logger: logger,
	}
}

func (r *Redis) CheckKeyInRedis(ctx context.Context, key string) error {
	getVal, err := r.GetFromRedis(ctx, key)
	if err == nil {
		switch getVal {
		case "":
			return fmt.Errorf("error in redis with key: no value for key %s", key)
		default:
			r.logger.With("key", key).Debug("alignment image already in redis")
			return nil
		}
	}
	return err
}

func (r *Redis) StroreInRedis(ctx context.Context, key, value string) error {
	check := r.CheckKeyInRedis(ctx, key)
	if check == redis.Nil {
		r.logger.With("key", key).Debug("Storing b64 image in redis")
		return r.client.Set(ctx, key, value, 60*time.Minute).Err()
	}
	return check
}

func (r *Redis) GetFromRedis(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}
