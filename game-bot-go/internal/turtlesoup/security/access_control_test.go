package security

import (
	"testing"

	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
)

func TestAccessControl_GetDenialReason(t *testing.T) {
	tests := []struct {
		name     string
		cfg      tsconfig.AccessConfig
		userID   string
		chatID   string
		wantDeny bool
		wantMsg  string
	}{
		{
			name:     "Passthrough",
			cfg:      tsconfig.AccessConfig{Passthrough: true},
			userID:   "user1",
			chatID:   "chat1",
			wantDeny: false,
		},
		{
			name:     "Blocked User",
			cfg:      tsconfig.AccessConfig{BlockedUserIDs: []string{"blocked_user"}},
			userID:   "blocked_user",
			chatID:   "chat1",
			wantDeny: true,
			wantMsg:  tsmessages.ErrorUserBlocked,
		},
		{
			name:     "Disabled (implicitly allowing all except blocked users)",
			cfg:      tsconfig.AccessConfig{Enabled: false},
			userID:   "user1",
			chatID:   "chat1",
			wantDeny: false,
		},
		{
			name:     "Blocked Chat",
			cfg:      tsconfig.AccessConfig{Enabled: true, BlockedChatIDs: []string{"blocked_chat"}},
			userID:   "user1",
			chatID:   "blocked_chat",
			wantDeny: true,
			wantMsg:  tsmessages.ErrorChatBlocked,
		},
		{
			name:     "Whitelisted Chat",
			cfg:      tsconfig.AccessConfig{Enabled: true, AllowedChatIDs: []string{"allowed_chat"}},
			userID:   "user1",
			chatID:   "allowed_chat",
			wantDeny: false,
		},
		{
			name:     "Non-Whitelisted Chat",
			cfg:      tsconfig.AccessConfig{Enabled: true, AllowedChatIDs: []string{"allowed_chat"}},
			userID:   "user1",
			chatID:   "other_chat",
			wantDeny: true,
			wantMsg:  tsmessages.ErrorAccessDenied,
		},
		{
			name:     "Empty Whitelist (Allow all if enabled but no whitelist)",
			cfg:      tsconfig.AccessConfig{Enabled: true, AllowedChatIDs: []string{}},
			userID:   "user1",
			chatID:   "chat1",
			wantDeny: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := NewAccessControl(tt.cfg)
			got := ac.GetDenialReason(tt.userID, tt.chatID)
			if tt.wantDeny {
				if got == nil {
					t.Errorf("expected denial, got permitted")
				} else if *got != tt.wantMsg {
					t.Errorf("expected denial message %q, got %q", tt.wantMsg, *got)
				}
			} else {
				if got != nil {
					t.Errorf("expected permitted, got denial: %v", *got)
				}
			}
		})
	}
}
