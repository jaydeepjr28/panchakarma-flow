package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
)

type clientLimit struct {
	count     int
	resetTime time.Time
}

// RateLimiter enforces a maximum request count per window per client IP.
func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*clientLimit)

	// Clean up stale entries every 2 minutes
	go func() {
		for {
			time.Sleep(2 * time.Minute)
			mu.Lock()
			now := time.Now()
			for ip, entry := range clients {
				if now.After(entry.resetTime) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		mu.Lock()
		entry, exists := clients[clientIP]
		if !exists || now.After(entry.resetTime) {
			clients[clientIP] = &clientLimit{
				count:     1,
				resetTime: now.Add(window),
			}
			mu.Unlock()
			c.Next()
			return
		}

		if entry.count >= maxRequests {
			retryAfter := int(entry.resetTime.Sub(now).Seconds())
			mu.Unlock()

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			appErr := errors.New(http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests. Please wait before trying again.", nil)
			response.Error(c, appErr)
			c.Abort()
			return
		}

		entry.count++
		mu.Unlock()
		c.Next()
	}
}
