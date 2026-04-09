package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// limiterEntry holds a rate limiter and the last time it was accessed,
// used to evict stale entries from the store.
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// limiterStore is a thread-safe map of key → limiter with background cleanup.
type limiterStore struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit
	burst    int
}

func newLimiterStore(r rate.Limit, burst int) *limiterStore {
	s := &limiterStore{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		burst:    burst,
	}
	go s.cleanup(5 * time.Minute)
	return s
}

func (s *limiterStore) get(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.limiters[key]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(s.r, s.burst)}
		s.limiters[key] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanup removes limiters that haven't been used within the given idle duration.
func (s *limiterStore) cleanup(idle time.Duration) {
	for {
		time.Sleep(idle)
		s.mu.Lock()
		for key, entry := range s.limiters {
			if time.Since(entry.lastSeen) > idle {
				delete(s.limiters, key)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimitByIP returns a middleware that limits requests per client IP.
// Intended for public (unauthenticated) endpoints.
// r is requests per second; burst allows short spikes above the steady rate.
func RateLimitByIP(r rate.Limit, burst int) gin.HandlerFunc {
	store := newLimiterStore(r, burst)
	return func(c *gin.Context) {
		if !store.get(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// RateLimitByUser returns a middleware that limits requests per authenticated user ID.
// Must be used after RequireAuth which sets the user_id in context.
// Falls back to IP-based limiting if no user ID is present.
func RateLimitByUser(r rate.Limit, burst int) gin.HandlerFunc {
	store := newLimiterStore(r, burst)
	return func(c *gin.Context) {
		key, exists := c.Get(string(ContextKeyUserID))
		if !exists || key == "" {
			key = c.ClientIP()
		}
		if !store.get(key.(string)).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
