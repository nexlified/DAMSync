package outbound

import (
	"context"
	"time"
)

// CachePort defines a generic key-value cache interface.
type CachePort interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	// GetBytes / SetBytes for binary data (transform cache).
	GetBytes(ctx context.Context, key string) ([]byte, error)
	SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// RateLimiterPort defines the interface for rate limiting.
type RateLimiterPort interface {
	// Allow checks if the key is within the limit and increments counter.
	// Returns (allowed, remaining, retryAfter, error).
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration, error)
}
