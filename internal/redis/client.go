package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/redis/go-redis/v9"
)

// Client wraps Redis client for Trading Token operations
type Client struct {
	client *redis.Client
}

// NewClient creates a new Redis client
func NewClient(cfg *config.Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info().
		Str("host", cfg.RedisHost).
		Int("port", cfg.RedisPort).
		Int("db", cfg.RedisDB).
		Msg("Redis connected successfully")

	return &Client{
		client: rdb,
	}, nil
}

// GetTradingToken retrieves trading token for a user from Redis
func (c *Client) GetTradingToken(ctx context.Context, userID int64) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	key := fmt.Sprintf("trading-token:%d", userID)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key doesn't exist
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetTradingToken stores trading token for a user in Redis with expiration
func (c *Client) SetTradingToken(ctx context.Context, userID int64, token string, expiresIn int) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	key := fmt.Sprintf("trading-token:%d", userID)
	// expiresIn is in seconds
	return c.client.Set(ctx, key, token, time.Duration(expiresIn)*time.Second).Err()
}

// Ping checks Redis connection
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return c.client.Ping(ctx).Err()
}

// Close closes Redis connection
func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}
