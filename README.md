## 常用限流算法 Golang 实现

### 支持的限流算法

| 算法        | 内存 | redis |
|-----------|----|----|
| 滑动窗口      | 支持 | 支持 |
| 令牌桶       | 支持 | 支持 |
| 漏斗        | 支持 | 支持 |
| 固定期限计数    | 支持 | 支持 |


### 示例

#### 1.滑动窗口

基于内存的滑动窗口限流器
```go
    l := slide_window.NewMemoryLimiter(context.Background(), &slide_window.MemorySlideWindowConfig{
        Rate:     50,
        Duration: time.Second,
        Size:     10,
    })
    ok, _ := l.Allow()
    if !ok {
        fmt.Println("Too Many Requests ")
    }
```

基于Redis的滑动窗口限流器
```go
    l := slide_window.NewRedisLimiter(context.Background(), &slide_window.RedisSlideWindowConfig{
        Rate:        50,
        Duration:    time.Second,
        RedisClient: getRedisClient(),
        Key:         "swlimit",
    })
    ok, _ := l.Allow()
    if !ok {
        fmt.Println("Too Many Requests ")
    }
```


#### 2.令牌桶

基于内存的令牌桶限流器
```go
    l := token_bucket.NewMemoryLimiter(context.Background(), &token_bucket.MemoryTokenBucketConfig{
        Rate:     10,
        Duration: time.Second,
    })
    ok, _ := l.Allow()
    if !ok {
        fmt.Println("Too Many Requests ")
    }
```

基于Redis的令牌桶限流器
```go
    l := token_bucket.NewRedisLimiter(context.Background(), &token_bucket.RedisTokenBucketConfig{
        Rate:        10,
        Duration:    time.Second,
        RedisClient: getRedisClient(),
        Key:         "tblimit",
    })
    ok, _ := l.Allow()
    if !ok {
        fmt.Println("Too Many Requests ")
    }
```


#### 3.漏斗

基于内存的漏斗限流器
```go
    l := leaky_bucket.NewMemoryLimiter(context.Background(), &leaky_bucket.MemoryLeakyBucketConfig{
        Rate:     10,
        Duration: time.Second,
    })
    ok, _ := l.Allow()
    if !ok {
        fmt.Println("Too Many Requests ")
    }
```

基于Redis的漏斗限流器
```go
    l := leaky_bucket.NewRedisLimiter(context.Background(), &leaky_bucket.RedisLeakyBucketConfig{
        Rate:        10,
        Duration:    time.Second,
        RedisClient: getRedisClient(),
        Key:         "lblimit",
    })
    ok, _ := l.Allow()
    if !ok {
        fmt.Println("Too Many Requests ")
    }
```


#### 4.固定期限计数

基于内存的固定期限计数限流器
```go
    l := fixed_period.NewMemoryLimiter(context.Background(), &fixed_period.MemoryFixedPeriodConfig{
        Rate:     20,
        Duration: time.Second,
    })
    ok, _ := l.Allow()
    if !ok {
        fmt.Println("Too Many Requests ")
    }
```

基于Redis的固定期限计数限流器
```go
    l := fixed_period.NewRedisLimiter(context.Background(), &fixed_period.RedisFixedPeriodConfig{
        Rate:        20,
        Duration:    time.Second,
        RedisClient: getRedisClient(),
        Key:         "fplimit",
    })
    ok, _ := l.Allow()
    if !ok {
        fmt.Println("Too Many Requests ")
    }
```

### TODO

- 优化
  - 锁优化，将互斥锁 sync.Mutex 调整为原子锁 atomic
  - redis 限流器优化，redis 异常时支持降级为内存限流
- 集成动态自适应限流算法
  - [kratos bbr](https://github.com/go-kratos/aegis/tree/main/ratelimit/bbr)
  - [go-zero load](https://github.com/zeromicro/go-zero/blob/cdd95296dbed0707d80c7aca74125c77dae3241e/core/load/adaptiveshedder.go)