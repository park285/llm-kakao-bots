package cache

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

type testPayload struct {
	Name string `json:"name"`
}

func newTestCacheService(t *testing.T) (*Service, *miniredis.Miniredis) {
	t.Helper()

	mini := miniredis.RunT(t)
	host, portStr, err := net.SplitHostPort(mini.Addr())
	if err != nil {
		t.Fatalf("failed to split address: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:       []string{net.JoinHostPort(host, portStr)},
		DisableCache:      true,
		ForceSingleClient: true,
	})
	if err != nil {
		t.Fatalf("failed to create valkey client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		t.Fatalf("failed to ping miniredis: %v", err)
	}
	svc := &Service{client: client, logger: logger}

	t.Cleanup(func() {
		_ = svc.Close()
		mini.Close()
	})

	return svc, mini
}

func TestCacheServiceSetGetAndExists(t *testing.T) {
	svc, mini := newTestCacheService(t)
	ctx := context.Background()

	value := testPayload{Name: "value"}
	if err := svc.Set(ctx, "key", value, 0); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	var got testPayload
	if err := svc.Get(ctx, "key", &got); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Name != "value" {
		t.Fatalf("unexpected value: %+v", got)
	}

	exists, err := svc.Exists(ctx, "key")
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected key to exist")
	}

	if err := svc.Expire(ctx, "key", time.Second); err != nil {
		t.Fatalf("expire failed: %v", err)
	}
	mini.FastForward(2 * time.Second)

	exists, err = svc.Exists(ctx, "key")
	if err != nil {
		t.Fatalf("exists after expire failed: %v", err)
	}
	if exists {
		t.Fatalf("expected key to expire")
	}
}

func TestCacheServiceMSetMGetDel(t *testing.T) {
	svc, _ := newTestCacheService(t)
	ctx := context.Background()

	pairs := map[string]any{
		"a": testPayload{Name: "A"},
		"b": testPayload{Name: "B"},
	}
	if err := svc.MSet(ctx, pairs, 0); err != nil {
		t.Fatalf("mset failed: %v", err)
	}

	values, err := svc.MGet(ctx, []string{"a", "b"})
	if err != nil {
		t.Fatalf("mget failed: %v", err)
	}
	var decoded testPayload
	if err := json.Unmarshal([]byte(values["a"]), &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded.Name != "A" {
		t.Fatalf("unexpected decoded value: %+v", decoded)
	}

	count, err := svc.DelMany(ctx, []string{"a", "b"})
	if err != nil {
		t.Fatalf("delmany failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 deletions, got %d", count)
	}
}

func TestMemberCacheOperations(t *testing.T) {
	svc, _ := newTestCacheService(t)
	ctx := context.Background()

	members := map[string]string{"member": "channel"}
	if err := svc.InitializeMemberDatabase(ctx, members); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	channelID, err := svc.GetMemberChannelID(ctx, "member")
	if err != nil {
		t.Fatalf("get member failed: %v", err)
	}
	if channelID != "channel" {
		t.Fatalf("unexpected channel id: %s", channelID)
	}

	all, err := svc.GetAllMembers(ctx)
	if err != nil {
		t.Fatalf("get all failed: %v", err)
	}
	if all["member"] != "channel" {
		t.Fatalf("unexpected members: %+v", all)
	}

	if err := svc.AddMember(ctx, "member2", "channel2"); err != nil {
		t.Fatalf("add member failed: %v", err)
	}
	channelID, err = svc.GetMemberChannelID(ctx, "member2")
	if err != nil {
		t.Fatalf("get member2 failed: %v", err)
	}
	if channelID != "channel2" {
		t.Fatalf("unexpected channel id: %s", channelID)
	}
}

func TestStreamCacheOperations(t *testing.T) {
	svc, _ := newTestCacheService(t)
	ctx := context.Background()

	streams := []*domain.Stream{{ID: "stream-1"}}
	svc.SetStreams(ctx, "streams:key", streams, time.Minute)

	got, found := svc.GetStreams(ctx, "streams:key")
	if !found || len(got) != 1 || got[0].ID != "stream-1" {
		t.Fatalf("unexpected streams: %+v, found=%v", got, found)
	}

	_, found = svc.GetStreams(ctx, "streams:missing")
	if found {
		t.Fatalf("expected missing streams to return false")
	}
}
