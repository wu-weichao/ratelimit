package slide_window

import (
	"context"
	"sync"
	"time"

	"github.com/wu-weichao/go-ratelimit"
)

// MemorySlideWindowLimiter 基于内存的滑动窗口限流器
type MemorySlideWindowLimiter struct {
	sync.Mutex

	rate     int           // 时间窗口内访问速率
	duration time.Duration // 时间窗口

	size        int           // 子窗口数量
	preDuration time.Duration // 子窗口时间
	buckets     []int         // 子窗口访问数
	last        time.Time     // 末次访问时间
}

// MemorySlideWindowConfig 基于内存滑动窗口限流器配置
type MemorySlideWindowConfig struct {
	Rate     int           // 窗口时间内访问速率
	Duration time.Duration // 窗口时间
	Size     int           // 子窗口数量
}

// NewMemoryLimiter .
func NewMemoryLimiter(ctx context.Context, conf *MemorySlideWindowConfig) ratelimit.Limiter {
	l := &MemorySlideWindowLimiter{
		rate:        conf.Rate,
		duration:    conf.Duration,
		size:        conf.Size,
		preDuration: conf.Duration / time.Duration(conf.Size),
		buckets:     make([]int, conf.Size),
		last:        time.Now(),
	}
	l.Reset()

	return l
}

// Allow 访问
func (l *MemorySlideWindowLimiter) Allow() (bool, error) {
	l.Lock()
	defer l.Unlock()

	// 访问时间
	now := time.Now()
	li := l.GetBucketIndex(l.last)
	ni := l.GetBucketIndex(now)
	// 子窗口存在变化
	if now.Sub(l.last) > (l.duration - l.preDuration) {
		// 超过一个窗口周期，重置所有子窗口
		l.Reset()
	} else if li != ni {
		// 在一个窗口周期内，重置区间子窗口
		l.ResetRange(li, ni)
	}
	// 访问受限
	if l.Count() >= l.rate {
		return false, nil
	}
	// 记录访问信息
	l.Incr(ni)
	l.last = now

	return true, nil
}

// GetBucketIndex 获取指定时间所属子窗口下标
func (l *MemorySlideWindowLimiter) GetBucketIndex(t time.Time) int {
	return int(t.UnixNano()/int64(l.preDuration)) % l.size
}

// Incr 给指定窗口访问数加1
func (l *MemorySlideWindowLimiter) Incr(i int) {
	l.buckets[i]++
}

// Count 窗口总访问数
func (l *MemorySlideWindowLimiter) Count() int {
	c := 0
	for i := 0; i < l.size; i++ {
		c += l.buckets[i]
	}
	return c
}

// Reset 重置所有子窗口访问数
func (l *MemorySlideWindowLimiter) Reset() {
	for i := 0; i < l.size; i++ {
		l.buckets[i] = 0
	}
}

// ResetRange 重置指定区间子窗口访问数，[左开，右闭)
func (l *MemorySlideWindowLimiter) ResetRange(start, end int) {
	i := start + 1
	for {
		// 超出，重置
		if i == l.size {
			i = 0
		}
		l.buckets[i] = 0
		// 中断
		if i == end {
			break
		}
		i++
	}
}
