package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	defaultQueueKey = "welcomebot:tasks"
)

// Client provides task queue operations.
type Client interface {
	Enqueue(ctx context.Context, task Task) error
	Dequeue(ctx context.Context, timeout time.Duration) (*Task, error)
	Close() error
}

// Task represents a queued task.
type Task struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	GuildID   string                 `json:"guild_id"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
	Retries   int                    `json:"retries"`
}

// Config contains queue configuration.
type Config struct {
	// Sentinel Configuration (preferred)
	SentinelAddrs []string // Sentinel addresses (e.g., ["sentinel1:26379", "sentinel2:26379"])
	MasterName    string   // Sentinel master name

	// Single Redis Configuration (fallback)
	RedisAddr string // Single Redis address (used if SentinelAddrs is empty)

	RedisPassword string
	RedisDB       int
	QueueKey      string
}

// DefaultConfig returns default queue configuration.
func DefaultConfig() Config {
	return Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       0,
		QueueKey:      defaultQueueKey,
	}
}

// redisQueue implements Client using Redis lists.
type redisQueue struct {
	client   *redis.Client
	queueKey string
}

// New creates a new queue client with the given configuration.
// Supports both Redis Sentinel (HA) and single Redis instance.
func New(cfg Config) (Client, error) {
	var rdb *redis.Client

	// Use Sentinel if configured
	if len(cfg.SentinelAddrs) > 0 && cfg.MasterName != "" {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.MasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.RedisPassword,
			DB:            cfg.RedisDB,
		})
	} else {
		// Fallback to single Redis
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	queueKey := cfg.QueueKey
	if queueKey == "" {
		queueKey = defaultQueueKey
	}

	return &redisQueue{
		client:   rdb,
		queueKey: queueKey,
	}, nil
}

// Enqueue adds a task to the queue.
func (q *redisQueue) Enqueue(ctx context.Context, task Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	if err := q.client.RPush(ctx, q.queueKey, data).Err(); err != nil {
		return fmt.Errorf("enqueue task %s: %w", task.ID, err)
	}

	return nil
}

// Dequeue removes and returns a task from the queue.
// Blocks until a task is available or timeout is reached.
func (q *redisQueue) Dequeue(ctx context.Context, timeout time.Duration) (*Task, error) {
	result, err := q.client.BLPop(ctx, timeout, q.queueKey).Result()
	if err == redis.Nil {
		return nil, nil // No task available
	}
	if err != nil {
		return nil, fmt.Errorf("dequeue task: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid blpop result")
	}

	var task Task
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}

	return &task, nil
}

// Close closes the queue client connection.
func (q *redisQueue) Close() error {
	if err := q.client.Close(); err != nil {
		return fmt.Errorf("close redis: %w", err)
	}
	return nil
}
