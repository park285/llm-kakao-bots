package processinglock

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/valkey-io/valkey-go"
)

func newTestService(t *testing.T, ttl time.Duration) (*Service, valkey.Client, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:       []string{mr.Addr()},
		DisableCache:      true,
		ForceSingleClient: true,
	})
	if err != nil {
		mr.Close()
		t.Fatalf("valkey client create failed: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := New(client, logger, func(chatID string) string {
		return "processing:" + chatID
	}, ttl)

	return svc, client, mr
}

func TestService_Start_IsMutualExclusion(t *testing.T) {
	svc, client, mr := newTestService(t, 10*time.Second)
	defer client.Close()
	defer mr.Close()

	ctx := context.Background()
	chatID := "room1"

	if err := svc.Start(ctx, chatID); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if err := svc.Start(ctx, chatID); !errors.Is(err, ErrAlreadyProcessing) {
		t.Fatalf("expected ErrAlreadyProcessing, got: %v", err)
	}

	ok, err := svc.IsProcessing(ctx, chatID)
	if err != nil {
		t.Fatalf("is processing failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected processing true")
	}

	if err := svc.Finish(ctx, chatID); err != nil {
		t.Fatalf("finish failed: %v", err)
	}

	ok, err = svc.IsProcessing(ctx, chatID)
	if err != nil {
		t.Fatalf("is processing failed: %v", err)
	}
	if ok {
		t.Fatalf("expected processing false")
	}

	if err := svc.Start(ctx, chatID); err != nil {
		t.Fatalf("start after finish failed: %v", err)
	}
}

func TestService_Start_TTLExpires(t *testing.T) {
	svc, client, mr := newTestService(t, 2*time.Second)
	defer client.Close()
	defer mr.Close()

	ctx := context.Background()
	chatID := "room1"

	if err := svc.Start(ctx, chatID); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	mr.FastForward(3 * time.Second)

	ok, err := svc.IsProcessing(ctx, chatID)
	if err != nil {
		t.Fatalf("is processing failed: %v", err)
	}
	if ok {
		t.Fatalf("expected processing false after ttl")
	}
}
