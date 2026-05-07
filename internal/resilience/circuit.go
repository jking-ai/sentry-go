package resilience

import (
	"context"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	Closed   State = iota // Normal — requests pass through.
	Open                  // Tripped — requests fail fast.
	HalfOpen              // Probing — allow one request to test recovery.
)

// CircuitBreaker prevents cascading failures by tripping open after
// consecutive failures and probing for recovery after a cooldown.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        State
	failures     int
	threshold    int
	cooldown     time.Duration
	lastFailure  time.Time
	onStateChange func(from, to State)
}

// NewCircuitBreaker creates a breaker that trips after threshold consecutive
// failures and stays open for cooldown before probing.
func NewCircuitBreaker(threshold int, cooldown time.Duration, opts ...func(*CircuitBreaker)) *CircuitBreaker {
	cb := &CircuitBreaker{
		state:     Closed,
		threshold: threshold,
		cooldown:  cooldown,
	}
	for _, o := range opts {
		o(cb)
	}
	return cb
}

// WithStateChangeHook registers a callback on state transitions.
func WithStateChangeHook(fn func(from, to State)) func(*CircuitBreaker) {
	return func(cb *CircuitBreaker) { cb.onStateChange = fn }
}

// Execute runs fn if the circuit allows it. It records success/failure and
// transitions the breaker state accordingly.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.allow(); err != nil {
		return err
	}

	err := fn()
	if err != nil {
		cb.recordFailure()
		return err
	}
	cb.recordSuccess()
	return nil
}

// allow checks whether a request can proceed.
func (cb *CircuitBreaker) allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return nil
	case Open:
		if time.Since(cb.lastFailure) > cb.cooldown {
			cb.transition(HalfOpen)
			return nil
		}
		return ErrCircuitOpen
	case HalfOpen:
		return nil
	}
	return nil
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.transition(Open)
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	if cb.state == HalfOpen {
		cb.transition(Closed)
	}
}

func (cb *CircuitBreaker) transition(to State) {
	from := cb.state
	if from == to {
		return
	}
	cb.state = to
	if cb.onStateChange != nil {
		cb.onStateChange(from, to)
	}
}

// State returns the current breaker state (for diagnostics).
func (cb *CircuitBreaker) CurrentState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// ErrCircuitOpen is returned when the circuit is open and requests are rejected.
var ErrCircuitOpen = &CircuitOpenError{}

// CircuitOpenError implements error.
type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string { return "circuit breaker is open" }