package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nexlified/dam/ports/outbound"
)

// NewRateLimiter returns a middleware that rate-limits by IP + API key.
func NewRateLimiter(rl outbound.RateLimiterPort, limit int, windowSecs int) fiber.Handler {
	window := time.Duration(windowSecs) * time.Second
	return func(c *fiber.Ctx) error {
		key := rateLimitKey(c)
		allowed, remaining, retryAfter, err := rl.Allow(c.Context(), key, limit, window)
		if err != nil {
			return c.Next() // allow on error
		}
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		if !allowed {
			c.Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter.Seconds(),
			})
		}
		return c.Next()
	}
}

func rateLimitKey(c *fiber.Ctx) string {
	// Prefer API key for keyed rate limits
	if key := c.Get("X-API-Key"); key != "" && len(key) > 8 {
		return "apikey:" + key[:8]
	}
	return "ip:" + c.IP()
}
