package consumer

import (
	"sync"
	"time"
)

// RateLimiter 是线程安全令牌桶，按总下载速度限制所有 goroutine。
type RateLimiter struct {
	mu         sync.Mutex
	rate       float64
	tokens     float64
	lastRefill time.Time
}

// NewRateLimiter 创建按字节/秒计的限速器。
func NewRateLimiter(rateBytesPerSecond int64) *RateLimiter {
	rate := float64(rateBytesPerSecond)
	return &RateLimiter{
		rate:       rate,
		tokens:     rate,
		lastRefill: time.Now(),
	}
}

// Acquire 在读取分块后阻塞，直到令牌足够。
func (r *RateLimiter) Acquire(numBytes int) {
	if r == nil || r.rate <= 0 {
		return
	}
	requestBytes := float64(numBytes)
	if requestBytes > r.rate {
		requestBytes = r.rate
	}
	for {
		r.mu.Lock()
		r.refillLocked()
		if r.tokens >= requestBytes {
			r.tokens -= requestBytes
			r.mu.Unlock()
			return
		}
		deficit := requestBytes - r.tokens
		r.mu.Unlock()
		time.Sleep(minDuration(time.Duration(deficit/r.rate*float64(time.Second)), 500*time.Millisecond))
	}
}

func (r *RateLimiter) refillLocked() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	r.tokens += elapsed * r.rate
	if r.tokens > r.rate {
		r.tokens = r.rate
	}
	r.lastRefill = now
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
