package fixed_period

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wu-weichao/go-ratelimit"
)

// redisFixedPeriodScript lua脚本
// KEYS[1] limitKey 限制的key
// ARGV[1] duration 窗口时长，单位：ms(毫秒)
// ARGV[2] rate 单位时间，限制次数
var redisFixedPeriodScript = `
local limitKey = KEYS[1]
local duration = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local total = redis.call("INCR", limitKey)
if total == 1 then
	redis.call("PEXPIREAT", limitKey, duration)
end
if total <= rate then
	return 1
else
	return 0
end
`

// RedisFixedPeriodLimiter 基于Redis的固定期限限流器
type RedisFixedPeriodLimiter struct {
	sync.Mutex

	rate     int           // 时间窗口内访问速率
	duration time.Duration // 时间窗口

	redisClient redis.Cmdable // redis 客户端
	key         string
	scriptSHA1  string
}

// RedisFixedPeriodConfig 基于Redis固定期限限流器配置
type RedisFixedPeriodConfig struct {
	Rate        int           // 窗口时间内访问速率
	Duration    time.Duration // 窗口时间
	RedisClient redis.Cmdable // Redis client
	Key         string        // 缓存 key
}

// NewRedisLimiter .
func NewRedisLimiter(ctx context.Context, conf *RedisFixedPeriodConfig) ratelimit.Limiter {
	// 计算脚本 sha1
	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(redisFixedPeriodScript)))

	l := &RedisFixedPeriodLimiter{
		rate:        conf.Rate,
		duration:    conf.Duration,
		redisClient: conf.RedisClient,
		key:         conf.Key,
		scriptSHA1:  scriptSHA1,
	}

	// 加载脚本
	if !l.redisClient.ScriptExists(ctx, l.scriptSHA1).Val()[0] {
		l.redisClient.ScriptLoad(ctx, redisFixedPeriodScript).Val()
	}
	return l
}

// Allow 访问
func (l *RedisFixedPeriodLimiter) Allow() (bool, error) {
	l.Lock()
	defer l.Unlock()

	// 窗口过期时间，毫秒
	expiredAt := time.Now().Add(l.duration).UnixMilli()
	ret, err := l.redisClient.EvalSha(context.Background(), l.scriptSHA1,
		[]string{l.key}, expiredAt, l.rate).Result()
	if err != nil {
		return false, err
	}
	if ret.(int64) == 0 {
		return false, nil
	}

	return true, nil
}
