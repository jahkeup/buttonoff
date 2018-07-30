package buttonoff

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Accepter interface {
	Accept(key string) bool
}

type pressRateLimiter struct {
	limit rate.Limit
	mu    *sync.RWMutex
	keys  map[string]*rate.Limiter
}

func NewPressRateLimiter(period time.Duration) *pressRateLimiter {
	limit := rate.Every(period)

	return &pressRateLimiter{
		limit: limit,
		mu:    &sync.RWMutex{},
		keys:  make(map[string]*rate.Limiter),
	}
}

func (press *pressRateLimiter) newLimiter() *rate.Limiter {
	// Burst of 1 allows the first invocation to use the token, bursting
	// as the first "allow".
	return rate.NewLimiter(press.limit, 1)
}

func (press *pressRateLimiter) Accept(key string) bool {
	press.mu.RLock()
	limiter, ok := press.keys[key]
	press.mu.RUnlock()
	if ok {
		return limiter.Allow()
	}

	// Double check.
	press.mu.Lock()
	defer press.mu.Unlock()
	limiter, ok = press.keys[key]
	if ok {
		return limiter.Allow()
	}

	// Or setup a new rate limiter.
	limiter = press.newLimiter()
	press.keys[key] = limiter

	return limiter.Allow()
}
