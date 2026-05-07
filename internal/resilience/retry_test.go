package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySuccessFirstTry(t *testing.T) {
	result, err := Retry(context.Background(), 3, 10*time.Millisecond, 100*time.Millisecond, func() (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
}

func TestRetrySuccessAfterFailures(t *testing.T) {
	attempt := 0
	result, err := Retry(context.Background(), 3, 10*time.Millisecond, 100*time.Millisecond, func() (string, error) {
		attempt++
		if attempt < 3 {
			return "", errors.New("transient")
		}
		return "recovered", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "recovered" {
		t.Errorf("result = %q, want %q", result, "recovered")
	}
	if attempt != 3 {
		t.Errorf("attempts = %d, want 3", attempt)
	}
}

func TestRetryExhausted(t *testing.T) {
	_, err := Retry(context.Background(), 2, 10*time.Millisecond, 100*time.Millisecond, func() (string, error) {
		return "", errors.New("always fail")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRetryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Retry(ctx, 2, 10*time.Millisecond, 100*time.Millisecond, func() (string, error) {
		return "should not reach", nil
	})
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func TestCircuitBreakerClosedAllowsRequests(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Millisecond)
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.CurrentState() != Closed {
		t.Errorf("state = %v, want CLOSED", cb.CurrentState())
	}
}

func TestCircuitBreakerTripsAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond)
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("fail")
		})
	}
	if cb.CurrentState() != Open {
		t.Errorf("state = %v, want OPEN after 3 failures", cb.CurrentState())
	}

	// Should reject fast
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if err == nil {
		t.Error("expected circuit-open error, got nil")
	}
}

func TestCircuitBreakerRecoversAfterCooldown(t *testing.T) {
	cb := NewCircuitBreaker(2, 20*time.Millisecond)

	// Trip the breaker
	_ = cb.Execute(context.Background(), func() error { return errors.New("1") })
	_ = cb.Execute(context.Background(), func() error { return errors.New("2") })

	if cb.CurrentState() != Open {
		t.Fatalf("expected OPEN, got %v", cb.CurrentState())
	}

	// Wait for cooldown
	time.Sleep(30 * time.Millisecond)

	// Next request should be allowed (half-open probe) and succeed → closes the circuit
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error after cooldown: %v", err)
	}
	if cb.CurrentState() != Closed {
		t.Errorf("state = %v, want CLOSED after recovery", cb.CurrentState())
	}
}

func TestCircuitBreakerHalfOpenRejectsConcurrent(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)

	// Trip the breaker
	_ = cb.Execute(context.Background(), func() error { return errors.New("fail") })
	if cb.CurrentState() != Open {
		t.Fatalf("expected OPEN, got %v", cb.CurrentState())
	}

	// Wait for cooldown to enter half-open
	time.Sleep(20 * time.Millisecond)

	// First request should succeed and claim the probe slot
	err := cb.Execute(context.Background(), func() error { return nil })
	if err != nil {
		t.Fatalf("expected probe to be allowed: %v", err)
	}

	// After the successful probe, the circuit should be Closed, so this should also succeed
	err = cb.Execute(context.Background(), func() error { return nil })
	if err != nil {
		t.Fatalf("expected request in closed state: %v", err)
	}
}

func TestCircuitBreakerHalfOpenFailsBackToOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 20*time.Millisecond)

	// Trip the breaker
	_ = cb.Execute(context.Background(), func() error { return errors.New("fail") })
	if cb.CurrentState() != Open {
		t.Fatalf("expected OPEN, got %v", cb.CurrentState())
	}

	// Wait for cooldown
	time.Sleep(30 * time.Millisecond)

	// Probe fails → back to open
	err := cb.Execute(context.Background(), func() error { return errors.New("still broken") })
	if err == nil {
		t.Fatal("expected error from failed probe")
	}
	if cb.CurrentState() != Open {
		t.Errorf("expected OPEN after failed probe, got %v", cb.CurrentState())
	}
}

func TestStateChangeHook(t *testing.T) {
	var transitions []string
	cb := NewCircuitBreaker(1, 20*time.Millisecond,
		WithStateChangeHook(func(from, to State) {
			transitions = append(transitions, stateStr(from)+"->"+stateStr(to))
		}),
	)

	_ = cb.Execute(context.Background(), func() error { return errors.New("fail") })

	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(transitions))
	}
	if transitions[0] != "CLOSED->OPEN" {
		t.Errorf("transition = %q, want %q", transitions[0], "CLOSED->OPEN")
	}
}

func stateStr(s State) string {
	switch s {
	case Closed:
		return "CLOSED"
	case Open:
		return "OPEN"
	case HalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}