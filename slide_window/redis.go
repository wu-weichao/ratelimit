package slide_window

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wu-weichao/go-ratelimit"
)

// redisSlideWindowScript lua脚本
// KEYS[1] limitKey 限制的key
// ARGV[1] window 窗口时长，单位：us(微秒)
// ARGV[2] rate 单位时间，限制次数
// ARGV[3] now 当前时间戳，单位：us(微秒)
var redisSlideWindowScript = `
local limitKey = KEYS[1]
local window = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local window_start = now - window
redis.call('ZREMRANGEBYSCORE', limitKey, '-inf', window_start)
local total = redis.call('ZCARD', KEYS[1])
if total < rate then
	redis.call('ZADD', limitKey, now, now)
	return 1
else
	return 0
end
`

// RedisSlideWindowLimiter 基于Redis的滑动窗口限流器
type RedisSlideWindowLimiter struct {
	sync.Mutex

	rate       int           // 时间窗口内访问速率
	duration   time.Duration // 时间窗口
	durationUs int64         // 窗口大小，us 微秒

	redisClient redis.Cmdable // redis 客户端
	key         string
	scriptSHA1  string
}

// RedisSlideWindowConfig 基于Redis滑动窗口限流器配置
type RedisSlideWindowConfig struct {
	Rate        int           // 窗口时间内访问速率
	Duration    time.Duration // 窗口时间
	RedisClient redis.Cmdable // Redis client
	Key         string        // 缓存 key
}

// NewRedisLimiter .
func NewRedisLimiter(ctx context.Context, conf *RedisSlideWindowConfig) ratelimit.Limiter {
	// 计算脚本 sha1
	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(redisSlideWindowScript)))

	l := &RedisSlideWindowLimiter{
		rate:        conf.Rate,
		duration:    conf.Duration,
		durationUs:  conf.Duration.Microseconds(),
		redisClient: conf.RedisClient,
		key:         conf.Key,
		scriptSHA1:  scriptSHA1,
	}

	// 加载脚本
	if !l.redisClient.ScriptExists(ctx, l.scriptSHA1).Val()[0] {
		l.redisClient.ScriptLoad(ctx, redisSlideWindowScript).Val()
	}
	return l
}

// Allow 访问
func (l *RedisSlideWindowLimiter) Allow() (bool, error) {
	l.Lock()
	defer l.Unlock()

	// 访问时间，单位微秒 us
	now := time.Now().UnixMicro()
	ret, err := l.redisClient.EvalSha(context.Background(), l.scriptSHA1,
		[]string{l.key}, l.durationUs, l.rate, now).Result()
	if err != nil {
		return false, err
	}
	if ret.(int64) == 0 {
		return false, nil
	}

	return true, nil
}
