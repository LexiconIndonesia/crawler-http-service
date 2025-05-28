package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/redis/go-redis/v9"
)

// Client represents a Redis client wrapper
type RedisClient struct {
	client *redis.Client
}

// NewClient creates a new Redis client instance
func NewClient(cfg config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
	}, nil
}

// Close closes the Redis client connection
func (c *RedisClient) Close() error {
	return c.client.Close()
}

// Set sets a key-value pair with optional expiration
func (c *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Delete removes a key
func (c *RedisClient) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (c *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}

// SetNX sets a key-value pair only if the key does not exist
func (c *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, expiration).Result()
}

// GetClient returns the underlying Redis client
func (c *RedisClient) GetClient() *redis.Client {
	return c.client
}
