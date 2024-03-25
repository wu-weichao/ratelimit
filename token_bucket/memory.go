package token_bucket

import (
	"context"
	"sync"
	"time"

	"github.com/wu-weichao/go-ratelimit"
)

// MemoryTokenBucketLimiter 基于内存的令牌桶限流器
type MemoryTokenBucketLimiter struct {
	sync.Mutex

	rate     int           // 时间窗口内访问速率
	duration time.Duration // 时间窗口

	tokens chan struct{}
}

// MemoryTokenBucketConfig 基于内存的令牌桶限流器配置
type MemoryTokenBucketConfig struct {
	Rate     int           // 窗口时间内访问速率
	Duration time.Duration // 窗口时间
}

// NewMemoryLimiter .
func NewMemoryLimiter(ctx context.Context, conf *MemoryTokenBucketConfig) ratelimit.Limiter {
	// 指定令牌桶容量
	tokens := make(chan struct{}, conf.Rate)

	l := &MemoryTokenBucketLimiter{
		rate:     conf.Rate,
		duration: conf.Duration,
		tokens:   tokens,
	}

	// 以固定速率生成令牌
	go func(l *MemoryTokenBucketLimiter) {
		ticker := time.NewTicker(conf.Duration / time.Duration(conf.Rate))
		for _ = range ticker.C {
			l.tokens <- struct{}{}
		}
	}(l)

	return l
}

// Allow 访问
func (l *MemoryTokenBucketLimiter) Allow() (bool, error) {
	select {
	case <-l.tokens:
		return true, nil
	default:
		return false, nil
	}
}
