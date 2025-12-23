package pending

import "testing"

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		entry    string
		wantOK   bool
		wantJSON string
	}{
		{
			name:     "valid format",
			entry:    "1702560000000|user123|{\"userId\":\"user123\",\"content\":\"hello\"}",
			wantOK:   true,
			wantJSON: "{\"userId\":\"user123\",\"content\":\"hello\"}",
		},
		{
			name:     "valid format with special chars in JSON",
			entry:    "1702560000000|user-456|{\"userId\":\"user-456\",\"content\":\"hello|world\"}",
			wantOK:   true,
			wantJSON: "{\"userId\":\"user-456\",\"content\":\"hello|world\"}",
		},
		{
			name:     "empty userId",
			entry:    "1702560000000||{\"content\":\"test\"}",
			wantOK:   true,
			wantJSON: "{\"content\":\"test\"}",
		},
		{
			name:     "missing second delimiter",
			entry:    "1702560000000|user123",
			wantOK:   false,
			wantJSON: "",
		},
		{
			name:     "missing first delimiter",
			entry:    "1702560000000user123|{}",
			wantOK:   false,
			wantJSON: "",
		},
		{
			name:     "empty string",
			entry:    "",
			wantOK:   false,
			wantJSON: "",
		},
		{
			name:     "only delimiters",
			entry:    "||",
			wantOK:   false,
			wantJSON: "",
		},
		{
			name:     "whitespace trimmed",
			entry:    "  1702560000000|user123|{\"test\":true}  ",
			wantOK:   true,
			wantJSON: "{\"test\":true}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotJSON, gotOK := ExtractJSON(tt.entry)
			if gotOK != tt.wantOK {
				t.Errorf("ExtractJSON() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotJSON != tt.wantJSON {
				t.Errorf("ExtractJSON() json = %q, want %q", gotJSON, tt.wantJSON)
			}
		})
	}
}

func TestEnqueueResult_String(t *testing.T) {
	tests := []struct {
		result EnqueueResult
		want   string
	}{
		{EnqueueSuccess, "SUCCESS"},
		{EnqueueQueueFull, "QUEUE_FULL"},
		{EnqueueDuplicate, "DUPLICATE"},
		{EnqueueResult(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.result.String(); got != tt.want {
				t.Errorf("EnqueueResult.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDequeueStatus_String(t *testing.T) {
	tests := []struct {
		status DequeueStatus
		want   string
	}{
		{DequeueEmpty, "EMPTY"},
		{DequeueExhausted, "EXHAUSTED"},
		{DequeueSuccess, "SUCCESS"},
		{DequeueStatus(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("DequeueStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("test:pending")

	if cfg.KeyPrefix != "test:pending" {
		t.Errorf("KeyPrefix = %v, want test:pending", cfg.KeyPrefix)
	}
	if cfg.MaxQueueSize != 5 {
		t.Errorf("MaxQueueSize = %v, want 5", cfg.MaxQueueSize)
	}
	if cfg.QueueTTLSeconds != 300 {
		t.Errorf("QueueTTLSeconds = %v, want 300", cfg.QueueTTLSeconds)
	}
	if cfg.StaleThresholdMS != 3600_000 {
		t.Errorf("StaleThresholdMS = %v, want 3600000", cfg.StaleThresholdMS)
	}
	if cfg.MaxDequeueIterations != 50 {
		t.Errorf("MaxDequeueIterations = %v, want 50", cfg.MaxDequeueIterations)
	}
}
