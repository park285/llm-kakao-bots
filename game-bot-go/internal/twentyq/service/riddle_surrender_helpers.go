package service

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func (s *RiddleService) buildSurrenderCategoryLine(category string) string {
	categoryKo := categoryToKorean(category)
	if categoryKo == nil {
		return ""
	}
	return s.msgProvider.Get(qmessages.SurrenderCategoryLine, messageprovider.P("category", *categoryKo))
}

func (s *RiddleService) buildSurrenderHintBlock(history []qmodel.QuestionHistory) string {
	for _, h := range history {
		if h.QuestionNumber < 0 {
			header := s.msgProvider.Get(qmessages.SurrenderHintBlockHeader, messageprovider.P("hintCount", 1))
			line := s.msgProvider.Get(qmessages.SurrenderHintItem, messageprovider.P("hintNumber", 1), messageprovider.P("content", h.Answer))
			return header + line
		}
	}
	return ""
}

func countHistoryStats(history []qmodel.QuestionHistory) (questionCount int, hintCount int) {
	for _, h := range history {
		if h.QuestionNumber > 0 {
			questionCount++
		}
		if h.QuestionNumber < 0 {
			hintCount++
		}
	}
	return questionCount, hintCount
}
