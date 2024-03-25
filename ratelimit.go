package ratelimit

// Limiter interface
type Limiter interface {
	Allow() (bool, error)
}
