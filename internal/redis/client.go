package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/redis/go-redis/v9"
)

// Client wraps Redis client for OTP and JWT token operations
type Client struct {
	client      *redis.Client
	otpKey      string
	otpTTL      time.Duration
	jwtTokenKey string
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
		client:      rdb,
		otpKey:      cfg.RedisOTPKey,
		otpTTL:      time.Duration(cfg.RedisOTPTTL) * time.Second,
		jwtTokenKey: cfg.RedisJWTKey,
	}, nil
}

// GetOTP retrieves OTP from Redis
func (c *Client) GetOTP(ctx context.Context) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	val, err := c.client.Get(ctx, c.otpKey).Result()
	if err == redis.Nil {
		return "", nil // Key doesn't exist
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetOTP stores OTP in Redis with TTL (atomic - replaces existing)
func (c *Client) SetOTP(ctx context.Context, otp string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return c.client.SetEx(ctx, c.otpKey, otp, c.otpTTL).Err()
}

// DeleteOTP removes OTP from Redis
func (c *Client) DeleteOTP(ctx context.Context) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return c.client.Del(ctx, c.otpKey).Err()
}

// GetOTPTTL gets remaining TTL for OTP
func (c *Client) GetOTPTTL(ctx context.Context) (time.Duration, error) {
	if c == nil || c.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}
	ttl, err := c.client.TTL(ctx, c.otpKey).Result()
	if err != nil {
		return 0, err
	}
	if ttl < 0 {
		return 0, nil // Key doesn't exist or has no expiry
	}
	return ttl, nil
}

// GetJWTToken retrieves JWT token from Redis
func (c *Client) GetJWTToken(ctx context.Context) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	val, err := c.client.Get(ctx, c.jwtTokenKey).Result()
	if err == redis.Nil {
		return "", nil // Key doesn't exist
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetJWTToken stores JWT token in Redis with no expiration (permanent)
func (c *Client) SetJWTToken(ctx context.Context, token string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return c.client.Set(ctx, c.jwtTokenKey, token, 0).Err()
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
