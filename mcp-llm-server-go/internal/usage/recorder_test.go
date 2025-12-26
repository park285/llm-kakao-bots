package usage

import (
	"fmt"
	"net"
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

func TestShouldFallbackToLocalhost(t *testing.T) {
	if shouldFallbackToLocalhost(nil, "postgres") {
		t.Fatalf("expected false for nil error")
	}
	if shouldFallbackToLocalhost(fmt.Errorf("no such host"), "") {
		t.Fatalf("expected false for empty host")
	}
	if shouldFallbackToLocalhost(fmt.Errorf("no such host"), "localhost") {
		t.Fatalf("expected false for localhost")
	}

	dnsErr := &net.DNSError{Name: "postgres"}
	if !shouldFallbackToLocalhost(dnsErr, "postgres") {
		t.Fatalf("expected fallback for dns error")
	}
	if !shouldFallbackToLocalhost(fmt.Errorf("wrapped: %w", dnsErr), "postgres") {
		t.Fatalf("expected fallback for wrapped dns error")
	}
	if shouldFallbackToLocalhost(dnsErr, "example") {
		t.Fatalf("expected false for non-postgres host")
	}

	if !shouldFallbackToLocalhost(fmt.Errorf("lookup postgres: no such host"), "postgres") {
		t.Fatalf("expected fallback for lookup failure")
	}
	if shouldFallbackToLocalhost(fmt.Errorf("no such host"), "postgres") {
		t.Fatalf("expected false when host name missing in error")
	}
}

func TestTodayDate(t *testing.T) {
	now := time.Now().In(time.Local)
	got := todayDate()
	if got.Location() != time.Local {
		t.Fatalf("unexpected location: %v", got.Location())
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
		t.Fatalf("expected midnight, got: %v", got)
	}

	expected := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !got.Equal(expected) && !got.Equal(expected.Add(24*time.Hour)) {
		t.Fatalf("unexpected date: %v", got)
	}
}
