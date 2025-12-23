package config

import (
	"testing"
	"time"
)

func TestIntFromEnv(t *testing.T) {
	key := "TEST_INT_ENV"

	t.Run("default", func(t *testing.T) {
		got, err := IntFromEnv(key, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 42 {
			t.Errorf("expected 42, got %d", got)
		}
	})

	t.Run("valid", func(t *testing.T) {
		t.Setenv(key, "100")
		got, err := IntFromEnv(key, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 100 {
			t.Errorf("expected 100, got %d", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv(key, "not_int")
		_, err := IntFromEnv(key, 42)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestBoolFromEnv(t *testing.T) {
	key := "TEST_BOOL_ENV"

	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
	}

	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			t.Setenv(key, tt.val)
			got, err := BoolFromEnv(key, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		t.Setenv(key, "maybe")
		_, err := BoolFromEnv(key, false)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestStringListFromEnv(t *testing.T) {
	key := "TEST_LIST_ENV"

	t.Setenv(key, "foo,bar, baz")
	got := StringListFromEnv(key, nil)
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	if got[0] != "foo" || got[1] != "bar" || got[2] != "baz" {
		t.Errorf("mismatch: %v", got)
	}

	t.Setenv(key, "  ")
	got = StringListFromEnv(key, []string{"default"})
	if len(got) != 1 || got[0] != "default" {
		t.Errorf("expected default, got %v", got)
	}
}

func TestDurationFromEnv(t *testing.T) {
	key := "TEST_DURATION_ENV"

	t.Setenv(key, "10")

	// Seconds
	d, err := DurationSecondsFromEnv(key, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 10*time.Second {
		t.Errorf("expected 10s, got %v", d)
	}

	// Millis
	d, err = DurationMillisFromEnv(key, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 10*time.Millisecond {
		t.Errorf("expected 10ms, got %v", d)
	}
}

func TestStringFromEnvFirstNonEmpty(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got := StringFromEnvFirstNonEmpty([]string{"TEST_FOO", "TEST_BAR"}, "default")
		if got != "default" {
			t.Errorf("expected default, got %q", got)
		}
	})

	t.Run("first_non_empty_wins", func(t *testing.T) {
		t.Setenv("TEST_FOO", "foo")
		t.Setenv("TEST_BAR", "bar")
		got := StringFromEnvFirstNonEmpty([]string{"TEST_FOO", "TEST_BAR"}, "default")
		if got != "foo" {
			t.Errorf("expected foo, got %q", got)
		}
	})

	t.Run("skips_empty", func(t *testing.T) {
		t.Setenv("TEST_FOO", "  ")
		t.Setenv("TEST_BAR", "bar")
		got := StringFromEnvFirstNonEmpty([]string{"TEST_FOO", "TEST_BAR"}, "default")
		if got != "bar" {
			t.Errorf("expected bar, got %q", got)
		}
	})
}

func TestIntFromEnvFirstNonEmpty(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got, err := IntFromEnvFirstNonEmpty([]string{"TEST_INT_FOO", "TEST_INT_BAR"}, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 7 {
			t.Errorf("expected 7, got %d", got)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		t.Setenv("TEST_INT_FOO", "  ")
		t.Setenv("TEST_INT_BAR", "10")
		got, err := IntFromEnvFirstNonEmpty([]string{"TEST_INT_FOO", "TEST_INT_BAR"}, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 10 {
			t.Errorf("expected 10, got %d", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv("TEST_INT_FOO", "oops")
		_, err := IntFromEnvFirstNonEmpty([]string{"TEST_INT_FOO", "TEST_INT_BAR"}, 7)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestInt64FromEnvFirstNonEmpty(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got, err := Int64FromEnvFirstNonEmpty([]string{"TEST_I64_FOO", "TEST_I64_BAR"}, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 7 {
			t.Errorf("expected 7, got %d", got)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		t.Setenv("TEST_I64_FOO", "  ")
		t.Setenv("TEST_I64_BAR", "10")
		got, err := Int64FromEnvFirstNonEmpty([]string{"TEST_I64_FOO", "TEST_I64_BAR"}, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 10 {
			t.Errorf("expected 10, got %d", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv("TEST_I64_FOO", "oops")
		_, err := Int64FromEnvFirstNonEmpty([]string{"TEST_I64_FOO", "TEST_I64_BAR"}, 7)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestBoolFromEnvFirstNonEmpty(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got, err := BoolFromEnvFirstNonEmpty([]string{"TEST_BOOL_FOO", "TEST_BOOL_BAR"}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != true {
			t.Errorf("expected true, got %v", got)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		t.Setenv("TEST_BOOL_FOO", "  ")
		t.Setenv("TEST_BOOL_BAR", "false")
		got, err := BoolFromEnvFirstNonEmpty([]string{"TEST_BOOL_FOO", "TEST_BOOL_BAR"}, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != false {
			t.Errorf("expected false, got %v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv("TEST_BOOL_FOO", "maybe")
		_, err := BoolFromEnvFirstNonEmpty([]string{"TEST_BOOL_FOO", "TEST_BOOL_BAR"}, true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestStringListFromEnvFirstNonEmpty(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got := StringListFromEnvFirstNonEmpty([]string{"TEST_LIST_FOO", "TEST_LIST_BAR"}, []string{"default"})
		if len(got) != 1 || got[0] != "default" {
			t.Errorf("expected default, got %v", got)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		t.Setenv("TEST_LIST_FOO", "  ")
		t.Setenv("TEST_LIST_BAR", "a,b, c")
		got := StringListFromEnvFirstNonEmpty([]string{"TEST_LIST_FOO", "TEST_LIST_BAR"}, nil)
		if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("unexpected list: %v", got)
		}
	})
}

func TestReadAccessConfigFromEnv(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg, err := ReadAccessConfigFromEnv(AccessConfigEnvOptions{
			EnvPrefix:             "TEST_",
			DefaultEnabled:        true,
			DefaultPassthrough:    false,
			DefaultAllowedChatIDs: []string{"1", "2"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Enabled != true {
			t.Errorf("expected Enabled=true, got %v", cfg.Enabled)
		}
		if cfg.Passthrough != false {
			t.Errorf("expected Passthrough=false, got %v", cfg.Passthrough)
		}
		if len(cfg.AllowedChatIDs) != 2 || cfg.AllowedChatIDs[0] != "1" || cfg.AllowedChatIDs[1] != "2" {
			t.Errorf("unexpected AllowedChatIDs: %v", cfg.AllowedChatIDs)
		}
	})

	t.Run("prefix_overrides_global", func(t *testing.T) {
		t.Setenv("ACCESS_ENABLED", "false")
		t.Setenv("TEST_ACCESS_ENABLED", "true")
		t.Setenv("ACCESS_PASSTHROUGH", "true")
		t.Setenv("TEST_ACCESS_PASSTHROUGH", "false")

		cfg, err := ReadAccessConfigFromEnv(AccessConfigEnvOptions{
			EnvPrefix:          "TEST_",
			DefaultEnabled:     false,
			DefaultPassthrough: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Enabled != true {
			t.Errorf("expected Enabled=true, got %v", cfg.Enabled)
		}
		if cfg.Passthrough != false {
			t.Errorf("expected Passthrough=false, got %v", cfg.Passthrough)
		}
	})

	t.Run("reads_lists", func(t *testing.T) {
		t.Setenv("TEST_ALLOWED_CHAT_IDS", "a,b")
		t.Setenv("TEST_BLOCKED_CHAT_IDS", "c")
		t.Setenv("TEST_BLOCKED_USER_IDS", "u1 u2")

		cfg, err := ReadAccessConfigFromEnv(AccessConfigEnvOptions{
			EnvPrefix:          "TEST_",
			DefaultEnabled:     false,
			DefaultPassthrough: false,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.AllowedChatIDs) != 2 || cfg.AllowedChatIDs[0] != "a" || cfg.AllowedChatIDs[1] != "b" {
			t.Errorf("unexpected AllowedChatIDs: %v", cfg.AllowedChatIDs)
		}
		if len(cfg.BlockedChatIDs) != 1 || cfg.BlockedChatIDs[0] != "c" {
			t.Errorf("unexpected BlockedChatIDs: %v", cfg.BlockedChatIDs)
		}
		if len(cfg.BlockedUserIDs) != 2 || cfg.BlockedUserIDs[0] != "u1" || cfg.BlockedUserIDs[1] != "u2" {
			t.Errorf("unexpected BlockedUserIDs: %v", cfg.BlockedUserIDs)
		}
	})
}

func TestReadServerTuningConfigFromEnv(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg, err := ReadServerTuningConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ReadHeaderTimeout != 5*time.Second {
			t.Errorf("expected ReadHeaderTimeout=5s, got %v", cfg.ReadHeaderTimeout)
		}
		if cfg.IdleTimeout != 0 {
			t.Errorf("expected IdleTimeout=0, got %v", cfg.IdleTimeout)
		}
		if cfg.MaxHeaderBytes != 0 {
			t.Errorf("expected MaxHeaderBytes=0, got %d", cfg.MaxHeaderBytes)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		t.Setenv("SERVER_READ_HEADER_TIMEOUT_SECONDS", "7")
		t.Setenv("SERVER_IDLE_TIMEOUT_SECONDS", "60")
		t.Setenv("SERVER_MAX_HEADER_BYTES", "8192")
		cfg, err := ReadServerTuningConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ReadHeaderTimeout != 7*time.Second {
			t.Errorf("expected ReadHeaderTimeout=7s, got %v", cfg.ReadHeaderTimeout)
		}
		if cfg.IdleTimeout != 60*time.Second {
			t.Errorf("expected IdleTimeout=60s, got %v", cfg.IdleTimeout)
		}
		if cfg.MaxHeaderBytes != 8192 {
			t.Errorf("expected MaxHeaderBytes=8192, got %d", cfg.MaxHeaderBytes)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv("SERVER_MAX_HEADER_BYTES", "-1")
		_, err := ReadServerTuningConfigFromEnv()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
