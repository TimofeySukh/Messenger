package room

import (
	"sync"
	"time"
)

type JoinLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewJoinLimiter(limit int, window time.Duration) *JoinLimiter {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}

	return &JoinLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (l *JoinLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	history := l.attempts[key]
	cutoff := now.Add(-l.window)
	filtered := history[:0]
	for _, t := range history {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= l.limit {
		l.attempts[key] = filtered
		return false
	}

	filtered = append(filtered, now)
	l.attempts[key] = filtered
	return true
}
