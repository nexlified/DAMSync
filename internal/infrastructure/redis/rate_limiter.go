package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *Client
}

func NewRateLimiter(client *Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow implements a sliding window rate limiter using Redis sorted sets.
// Returns: allowed, remaining, retryAfter, error
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	rkey := fmt.Sprintf("rl:%s", key)

	pipe := rl.client.Raw().Pipeline()
	pipe.ZRemRangeByScore(ctx, rkey, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
	countCmd := pipe.ZCard(ctx, rkey)
	member := fmt.Sprintf("%d:%s", now.UnixNano(), rkey)
	pipe.ZAdd(ctx, rkey, goredis.Z{Score: float64(now.UnixMilli()), Member: member})
	pipe.Expire(ctx, rkey, window+time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return true, limit, 0, err
	}

	count := int(countCmd.Val())
	remaining := limit - count - 1

	if count >= limit {
		oldest, err := rl.client.Raw().ZRangeWithScores(ctx, rkey, 0, 0).Result()
		if err != nil || len(oldest) == 0 {
			return false, 0, window, nil
		}
		oldestTime := time.UnixMilli(int64(oldest[0].Score))
		retryAfter := window - time.Since(oldestTime)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, 0, retryAfter, nil
	}

	if remaining < 0 {
		remaining = 0
	}
	return true, remaining, 0, nil
}
