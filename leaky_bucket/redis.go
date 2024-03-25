package leaky_bucket

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wu-weichao/go-ratelimit"
)

// redisLeakyBucketScript lua脚本
// KEYS[1] limitKey 限制的key
// ARGV[1] interval 访问速率间隔，单位：us(微秒)
// ARGV[2] now 当前时间戳，单位：us(微秒)
var redisLeakyBucketScript = `
local limitKey = KEYS[1]
local interval = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local last = redis.call("GET", limitKey)
if last == false then
    last = 0
end
local next = tonumber(last) + interval
if now > next then
	redis.call("SET", limitKey, now)
	return 1
else
	return 0
end
`

// RedisLeakyBucketLimiter 基于Redis的漏桶限流器
type RedisLeakyBucketLimiter struct {
	sync.Mutex

	rate     int           // 时间窗口内访问速率
	duration time.Duration // 时间窗口
	interval int64         // 访问速率间隔，us 微秒

	redisClient redis.Cmdable // redis 客户端
	key         string
	scriptSHA1  string
}

// RedisLeakyBucketConfig 基于Redis漏桶限流器配置
type RedisLeakyBucketConfig struct {
	Rate        int           // 窗口时间内访问速率
	Duration    time.Duration // 窗口时间
	RedisClient redis.Cmdable // Redis client
	Key         string        // 缓存 key
}

// NewRedisLimiter .
func NewRedisLimiter(ctx context.Context, conf *RedisLeakyBucketConfig) ratelimit.Limiter {
	// 计算脚本 sha1
	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(redisLeakyBucketScript)))

	l := &RedisLeakyBucketLimiter{
		rate:        conf.Rate,
		duration:    conf.Duration,
		interval:    (conf.Duration / time.Duration(conf.Rate)).Microseconds(),
		redisClient: conf.RedisClient,
		key:         conf.Key,
		scriptSHA1:  scriptSHA1,
	}

	// 加载脚本
	if !l.redisClient.ScriptExists(ctx, l.scriptSHA1).Val()[0] {
		l.redisClient.ScriptLoad(ctx, redisLeakyBucketScript).Val()
	}
	return l
}

// Allow 访问
func (l *RedisLeakyBucketLimiter) Allow() (bool, error) {
	l.Lock()
	defer l.Unlock()

	// 访问时间，单位微秒 us
	now := time.Now().UnixMicro()
	ret, err := l.redisClient.EvalSha(context.Background(), l.scriptSHA1,
		[]string{l.key}, l.interval, now).Result()
	if err != nil {
		return false, err
	}
	if ret.(int64) == 0 {
		return false, nil
	}

	return true, nil
}
