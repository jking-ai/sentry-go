package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// Default retry configuration.
const (
	DefaultMaxRetries = 3
	DefaultBaseDelay  = 500 * time.Millisecond
	DefaultMaxDelay   = 10 * time.Second
)

// Result holds the outcome of a retry operation.
type Result[T any] struct {
	Value T
	Err   error
}

// Retry executes fn with exponential backoff + jitter. Returns the value on
// success or the last error after all attempts are exhausted.
func Retry[T any](ctx context.Context, maxRetries int, baseDelay, maxDelay time.Duration, fn func() (T, error)) (T, error) {
	var lastErr error
	var zero T

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		val, err := fn()
		if err == nil {
			return val, nil
		}
		lastErr = err

		if attempt == maxRetries {
			break
		}

		delay := backoff(attempt, baseDelay, maxDelay)
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	return zero, fmt.Errorf("resilience: all %d attempts failed: %w", maxRetries+1, lastErr)
}

// backoff computes exponential backoff with full jitter.
func backoff(attempt int, base, max time.Duration) time.Duration {
	exp := math.Pow(2, float64(attempt))
	delay := float64(base) * exp
	jitter := rand.Float64() * delay // full jitter
	d := time.Duration(jitter)
	if d > max {
		d = max
	}
	if d < base {
		d = base
	}
	return d
}