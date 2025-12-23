package usage

import (
	"testing"
	"time"
)

func TestBatcherBackoff(t *testing.T) {
	b := &batcher{flushInterval: time.Second, maxBackoff: 4 * time.Second}

	b.consecutiveFlushFailures = 1
	if backoff := b.computeBackoff(); backoff != time.Second {
		t.Fatalf("unexpected backoff: %v", backoff)
	}

	b.consecutiveFlushFailures = 2
	if backoff := b.computeBackoff(); backoff != 2*time.Second {
		t.Fatalf("unexpected backoff: %v", backoff)
	}

	b.consecutiveFlushFailures = 3
	if backoff := b.computeBackoff(); backoff != 4*time.Second {
		t.Fatalf("unexpected backoff: %v", backoff)
	}

	b.consecutiveFlushFailures = 4
	if backoff := b.computeBackoff(); backoff != 4*time.Second {
		t.Fatalf("unexpected backoff cap: %v", backoff)
	}
}

func TestBatcherShouldLogFailure(t *testing.T) {
	b := &batcher{errorLogMaxInterval: time.Hour}
	b.consecutiveFlushFailures = 1
	if !b.shouldLogFailure() {
		t.Fatalf("expected log on first failure")
	}

	b.consecutiveFlushFailures = 3
	b.lastErrorLoggedAt = time.Now()
	if b.shouldLogFailure() {
		t.Fatalf("did not expect log for non power-of-two")
	}
}

func TestIsPowerOfTwo(t *testing.T) {
	if !isPowerOfTwo(1) || !isPowerOfTwo(2) || !isPowerOfTwo(4) {
		t.Fatalf("expected power of two")
	}
	if isPowerOfTwo(3) || isPowerOfTwo(0) {
		t.Fatalf("unexpected power of two")
	}
}

func TestMaxInt(t *testing.T) {
	if maxInt(1, 2) != 2 {
		t.Fatalf("unexpected max")
	}
}
