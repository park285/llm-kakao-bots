package models

import "strings"

// DisplayName: 표시용 이름을 계산합니다.
func DisplayName(chatID string, userID string, sender *string, anonymous string) string {
	if sender != nil && strings.TrimSpace(*sender) != "" {
		return strings.TrimSpace(*sender)
	}
	userID = strings.TrimSpace(userID)
	chatID = strings.TrimSpace(chatID)
	if userID == "" || chatID == "" || userID == chatID {
		return anonymous
	}
	return userID
}

// DisplayNameFromUser: 유저 정보로 표시용 이름을 계산합니다.
func DisplayNameFromUser(userID string, sender *string) string {
	if sender != nil && strings.TrimSpace(*sender) != "" {
		return strings.TrimSpace(*sender)
	}
	return strings.TrimSpace(userID)
}
