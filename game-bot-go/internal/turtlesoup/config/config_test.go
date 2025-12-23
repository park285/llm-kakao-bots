package config

import (
	"testing"
	"time"
)

func TestReadInjectionGuardConfig(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg, err := readInjectionGuardConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CacheTTL != 600*time.Second {
			t.Errorf("expected CacheTTL=600s, got %v", cfg.CacheTTL)
		}
		if cfg.CacheMaxEntries != 10000 {
			t.Errorf("expected CacheMaxEntries=10000, got %d", cfg.CacheMaxEntries)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		t.Setenv("TURTLESOUP_INJECTION_GUARD_CACHE_TTL_SECONDS", "30")
		t.Setenv("TURTLESOUP_INJECTION_GUARD_CACHE_MAX_ENTRIES", "123")
		cfg, err := readInjectionGuardConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CacheTTL != 30*time.Second {
			t.Errorf("expected CacheTTL=30s, got %v", cfg.CacheTTL)
		}
		if cfg.CacheMaxEntries != 123 {
			t.Errorf("expected CacheMaxEntries=123, got %d", cfg.CacheMaxEntries)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv("TURTLESOUP_INJECTION_GUARD_CACHE_TTL_SECONDS", "-1")
		_, err := readInjectionGuardConfig()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

