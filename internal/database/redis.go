package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedis creates a Redis connection by building a dsn out of config.RedisConfig
// and checks for connectivity by pinging.
// Returns an error if the DSN is wrong or the app can't ping the Redis instance.
func NewRedis(host string, port int, password string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		if err := client.Close(); err != nil {
			return nil, fmt.Errorf("failed to close redis client: %w", err)
		}
		return nil, fmt.Errorf("failed to ping redis client: %w", err)
	}

	return client, nil
}
