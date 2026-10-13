package blacklist

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "comets.jwt."

type Blacklist struct {
	client *redis.Client
}

func FromEnv(ctx context.Context) (*Blacklist, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password:    os.Getenv("REDIS_PASSWORD"),
		DialTimeout: 5 * time.Second,
		ReadTimeout: 5 * time.Second,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Blacklist{client: client}, nil
}

func (b *Blacklist) Add(ctx context.Context, token, login string, ttl time.Duration) error {
	return b.client.Set(ctx, keyPrefix+token, login, ttl).Err()
}

func (b *Blacklist) Contains(ctx context.Context, token string) (bool, error) {
	err := b.client.Get(ctx, keyPrefix+token).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	return err == nil, err
}
