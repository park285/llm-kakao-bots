package service

import (
	"context"
	"strings"
)

func (s *RiddleService) cleanupSession(ctx context.Context, chatID string) {
	players, err := s.playerStore.GetAll(ctx, chatID)
	userIDs := make([]string, 0, len(players))
	if err == nil {
		for _, p := range players {
			if strings.TrimSpace(p.UserID) == "" {
				continue
			}
			userIDs = append(userIDs, p.UserID)
		}
	}

	_ = s.sessionStore.Delete(ctx, chatID)
	_ = s.historyStore.Clear(ctx, chatID)
	_ = s.hintCountStore.Delete(ctx, chatID)
	_ = s.categoryStore.Save(ctx, chatID, nil)
	_ = s.playerStore.Clear(ctx, chatID)
	_ = s.wrongGuessStore.Delete(ctx, chatID, userIDs)
	_ = s.voteStore.Clear(ctx, chatID)
}
