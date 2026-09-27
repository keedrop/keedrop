package main

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultRateLimit = 60
	rateLimitWindow  = time.Minute
)

// counts requests per client IP in fixed one-minute windows.
// All counters are dropped when a window ends, so memory stays bounded
// by the number of clients seen within a single window.
type rateLimiter struct {
	mu          sync.Mutex
	limit       int
	windowStart time.Time
	counts      map[string]int
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{limit: limit, windowStart: time.Now(), counts: make(map[string]int)}
}

func (l *rateLimiter) allow(client string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now := time.Now(); now.Sub(l.windowStart) >= rateLimitWindow {
		l.windowStart = now
		l.counts = make(map[string]int)
	}
	l.counts[client]++
	return l.counts[client] <= l.limit
}

func (l *rateLimiter) middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !l.allow(ctx.ClientIP()) {
			ctx.Header("Retry-After", strconv.Itoa(int(rateLimitWindow.Seconds())))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}
		ctx.Next()
	}
}

// requests per minute and client, 0 disables rate limiting
func rateLimit() int {
	if value := os.Getenv("KEEDROP_RATE_LIMIT"); len(value) > 0 {
		if limit, err := strconv.Atoi(value); err == nil && limit >= 0 {
			return limit
		}
		logger.Warning("Ignoring invalid KEEDROP_RATE_LIMIT:", value)
	}
	return defaultRateLimit
}
