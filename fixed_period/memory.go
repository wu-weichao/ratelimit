package fixed_period

import (
	"context"
	"sync"
	"time"

	"github.com/wu-weichao/go-ratelimit"
)

// MemoryFixedPeriodLimiter 基于内存的固定期限限流器
type MemoryFixedPeriodLimiter struct {
	sync.Mutex

	rate     int           // 时间窗口内访问速率
	duration time.Duration // 时间窗口

	count     int       // 时间窗口内访问次数
	expiredAt time.Time // 过期时间
}

// MemoryFixedPeriodConfig 基于内存的固定期限限流器配置
type MemoryFixedPeriodConfig struct {
	Rate     int           // 窗口时间内访问速率
	Duration time.Duration // 窗口时间
}

// NewMemoryLimiter .
func NewMemoryLimiter(ctx context.Context, conf *MemoryFixedPeriodConfig) ratelimit.Limiter {
	l := &MemoryFixedPeriodLimiter{
		rate:      conf.Rate,
		duration:  conf.Duration,
		expiredAt: time.Now().Add(conf.Duration),
	}

	return l
}

// Allow 访问
func (l *MemoryFixedPeriodLimiter) Allow() (bool, error) {
	l.Lock()
	defer l.Unlock()

	// 访问时间
	now := time.Now()
	// 过期，重置窗口
	if now.After(l.expiredAt) {
		l.count = 0
		l.expiredAt = now.Add(l.duration)
	}
	// 超出限制
	if l.count >= l.rate {
		return false, nil
	}
	l.count++

	return true, nil
}
