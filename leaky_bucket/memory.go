package leaky_bucket

import (
	"context"
	"sync"
	"time"

	"github.com/wu-weichao/go-ratelimit"
)

// MemoryLeakyBucketLimiter 基于内存的漏桶限流器
type MemoryLeakyBucketLimiter struct {
	sync.Mutex

	rate     int           // 时间窗口内访问速率
	duration time.Duration // 时间窗口

	interval time.Duration // 访问速率间隔，us 微秒
	last     time.Time     // 末次访问时间
}

// MemoryLeakyBucketConfig 基于内存的漏桶限流器配置
type MemoryLeakyBucketConfig struct {
	Rate     int           // 窗口时间内访问速率
	Duration time.Duration // 窗口时间
}

// NewMemoryLimiter .
func NewMemoryLimiter(ctx context.Context, conf *MemoryLeakyBucketConfig) ratelimit.Limiter {
	return &MemoryLeakyBucketLimiter{
		rate:     conf.Rate,
		duration: conf.Duration,
		interval: conf.Duration / time.Duration(conf.Rate),
		last:     time.Now(),
	}
}

// Allow 访问
// 只控制访问速率
func (l *MemoryLeakyBucketLimiter) Allow() (bool, error) {
	l.Lock()
	defer l.Unlock()

	// 访问时间
	now := time.Now()
	// 访问受限
	if now.Sub(l.last) < l.interval {
		return false, nil
	}
	// 记录访问信息
	l.last = now

	return true, nil
}
