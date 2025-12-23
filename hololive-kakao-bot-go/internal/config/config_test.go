package config

import (
	"reflect"
	"testing"
)

func TestCollectAPIKeys(t *testing.T) {
	prefix := "HOLODEX_API_KEY_"

	t.Setenv("HOLODEX_API_KEY_1", " key-1 ")
	t.Setenv("HOLODEX_API_KEY_2", "key-2")
	t.Setenv("HOLODEX_API_KEY_3", "key-3")
	t.Setenv("HOLODEX_API_KEY_4", "key-4")
	t.Setenv("HOLODEX_API_KEY_5", "key-5")
	t.Setenv("HOLODEX_API_KEYS", "key-2,key-6 , key-7")

	keys := collectAPIKeys(prefix)

	expected := []string{"key-1", "key-2", "key-3", "key-4", "key-5", "key-6", "key-7"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("collectAPIKeys() = %v, expected %v", keys, expected)
	}
}

func TestKakaoConfig_IsRoomAllowed(t *testing.T) {
	t.Run("ACL disabled allows all", func(t *testing.T) {
		cfg := KakaoConfig{
			Rooms:      []string{"room-a"},
			ACLEnabled: false,
		}

		if !cfg.IsRoomAllowed("other-room", "999") {
			t.Fatalf("expected room to be allowed when ACL is disabled")
		}
	})

	t.Run("Matches by chat ID only", func(t *testing.T) {
		cfg := KakaoConfig{
			Rooms:      []string{"1234567890"},
			ACLEnabled: true,
		}

		// chatID가 일치하면 허용
		if !cfg.IsRoomAllowed("테스트방", "1234567890") {
			t.Fatalf("expected room to be allowed by chat ID")
		}

		// roomName만 일치해도 chatID가 다르면 거부
		if cfg.IsRoomAllowed("1234567890", "other-id") {
			t.Fatalf("expected room to be denied - only chatID should be checked")
		}
	})

	t.Run("Empty chatID denies", func(t *testing.T) {
		cfg := KakaoConfig{
			Rooms:      []string{"테스트방"},
			ACLEnabled: true,
		}

		// chatID가 비어있으면 거부
		if cfg.IsRoomAllowed("테스트방", "") {
			t.Fatalf("expected room to be denied when chatID is empty")
		}
	})

	t.Run("No match denies", func(t *testing.T) {
		cfg := KakaoConfig{
			Rooms:      []string{"allowed-room"},
			ACLEnabled: true,
		}

		if cfg.IsRoomAllowed("other-room", "999") {
			t.Fatalf("expected room to be denied when no match exists")
		}
	})
}

func TestKakaoConfig_AddRemoveRoom(t *testing.T) {
	cfg := KakaoConfig{
		Rooms:      []string{"123"},
		ACLEnabled: true,
	}

	if !cfg.AddRoom(" 456 ") {
		t.Fatalf("expected AddRoom to succeed")
	}
	if cfg.AddRoom("456") {
		t.Fatalf("expected duplicate AddRoom to fail")
	}

	if !cfg.RemoveRoom(" 456 ") {
		t.Fatalf("expected RemoveRoom to succeed")
	}
	if cfg.RemoveRoom("456") {
		t.Fatalf("expected RemoveRoom to fail for non-existing room")
	}
}

func TestKakaoConfig_SnapshotACL_ReturnsCopy(t *testing.T) {
	cfg := KakaoConfig{
		Rooms:      []string{"a"},
		ACLEnabled: true,
	}

	enabled, rooms := cfg.SnapshotACL()
	if !enabled {
		t.Fatalf("expected enabled to be true")
	}
	if len(rooms) != 1 || rooms[0] != "a" {
		t.Fatalf("unexpected rooms snapshot: %v", rooms)
	}

	rooms[0] = "mutated"
	_, rooms2 := cfg.SnapshotACL()
	if rooms2[0] != "a" {
		t.Fatalf("expected SnapshotACL to return a copy, got: %v", rooms2)
	}
}
