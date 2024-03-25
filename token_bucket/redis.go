package token_bucket

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wu-weichao/go-ratelimit"
)

// redisTokenBucketScript lua脚本
// KEYS[1] limitKey 限制的key
// ARGV[1] duration  窗口时长，单位：us(微秒)
// ARGV[2] rate 单位时间，限制次数
// ARGV[3] now 当前时间戳，单位：us(微秒)
// ARGV[4] requested 请求令牌数
var redisTokenBucketScript = `
local limitKey = KEYS[1]
local duration = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])
local lastTs = redis.call("HGET", limitKey, "ts")
if lastTs == false then
    lastTs = 0
else
	lastTs = tonumber(lastTs)
end
local lastToken = redis.call("HGET", limitKey, "token")
if lastToken == false then
    lastToken = 0
end
local interval = duration / rate
local diffTs = math.max(0, now - lastTs)
local fillToken = math.min(rate,  tonumber(lastToken) + diffTs / interval)
if fillToken < requested then
	return 0
end
redis.call("HSET", limitKey, "ts", now)
redis.call("HSET", limitKey, "token", fillToken - requested)
return 1
`

// RedisTokenBucketLimiter 基于Redis的令牌桶限流器
type RedisTokenBucketLimiter struct {
	sync.Mutex

	rate       int           // 时间窗口内访问速率
	duration   time.Duration // 时间窗口
	durationUs int64         // 窗口大小，us 微秒

	redisClient redis.Cmdable // redis 客户端
	key         string
	scriptSHA1  string
}

// RedisTokenBucketConfig 基于Redis令牌桶限流器配置
type RedisTokenBucketConfig struct {
	Rate        int           // 窗口时间内访问速率
	Duration    time.Duration // 窗口时间
	RedisClient redis.Cmdable // Redis client
	Key         string        // 缓存 key
}

// NewRedisLimiter .
func NewRedisLimiter(ctx context.Context, conf *RedisTokenBucketConfig) ratelimit.Limiter {
	// 计算脚本 sha1
	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(redisTokenBucketScript)))

	l := &RedisTokenBucketLimiter{
		rate:        conf.Rate,
		duration:    conf.Duration,
		durationUs:  conf.Duration.Microseconds(),
		redisClient: conf.RedisClient,
		key:         conf.Key,
		scriptSHA1:  scriptSHA1,
	}

	// 加载脚本
	if !l.redisClient.ScriptExists(ctx, l.scriptSHA1).Val()[0] {
		l.redisClient.ScriptLoad(ctx, redisTokenBucketScript).Val()
	}
	return l
}

// Allow 访问
func (l *RedisTokenBucketLimiter) Allow() (bool, error) {
	return l.AllowN(1)
}

// AllowN 访问
func (l *RedisTokenBucketLimiter) AllowN(n int) (bool, error) {
	l.Lock()
	defer l.Unlock()

	// 访问时间，单位微秒 us
	now := time.Now().UnixMicro()
	ret, err := l.redisClient.EvalSha(context.Background(), l.scriptSHA1,
		[]string{l.key}, l.durationUs, l.rate, now, n).Result()
	if err != nil {
		return false, err
	}
	if ret.(int64) == 0 {
		return false, nil
	}

	return true, nil
}
