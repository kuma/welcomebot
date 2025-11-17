package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Client provides caching operations.
type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetJSON(ctx context.Context, key string, dest interface{}) error
	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Close() error
}

// Config contains Redis configuration.
type Config struct {
	// Sentinel Configuration (preferred)
	SentinelAddrs  []string // Sentinel addresses (e.g., ["sentinel1:26379", "sentinel2:26379"])
	MasterName     string   // Sentinel master name
	
	// Single Redis Configuration (fallback)
	Addr     string // Single Redis address (used if SentinelAddrs is empty)
	
	Password string
	DB       int
}

// DefaultConfig returns default cache configuration.
func DefaultConfig() Config {
	return Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
}

// redisClient implements Client using Redis.
type redisClient struct {
	client *redis.Client
}

// New creates a new cache client with the given configuration.
// Supports both Redis Sentinel (HA) and single Redis instance.
func New(cfg Config) (Client, error) {
	var rdb *redis.Client
	
	// Use Sentinel if configured
	if len(cfg.SentinelAddrs) > 0 && cfg.MasterName != "" {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.MasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.Password,
			DB:            cfg.DB,
		})
	} else {
		// Fallback to single Redis
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &redisClient{client: rdb}, nil
}

// Get retrieves a value from the cache.
func (c *redisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("get key %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value in the cache with the given TTL.
func (c *redisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("set key %s: %w", key, err)
	}
	return nil
}

// Delete removes a key from the cache.
func (c *redisClient) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete key %s: %w", key, err)
	}
	return nil
}

// Exists checks if a key exists in the cache.
func (c *redisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check exists %s: %w", key, err)
	}
	return count > 0, nil
}

// GetJSON retrieves and unmarshals JSON from the cache.
func (c *redisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("unmarshal json for key %s: %w", key, err)
	}

	return nil
}

// SetJSON marshals and stores JSON in the cache.
func (c *redisClient) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal json for key %s: %w", key, err)
	}

	return c.Set(ctx, key, string(data), ttl)
}

// Close closes the cache client connection.
func (c *redisClient) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close redis: %w", err)
	}
	return nil
}

