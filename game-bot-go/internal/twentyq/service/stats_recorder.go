package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qrepo "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
)

// GameResult: 게임 결과 타입 정의
type GameResult string

// GameResultCorrect: 게임 결과 상수 목록입니다.
const (
	GameResultCorrect   GameResult = "CORRECT"
	GameResultSurrender GameResult = "SURRENDER"
)

// PlayerCompletionRecord: 플레이어별 완료 기록 구조체
type PlayerCompletionRecord struct {
	UserID          string
	Sender          string
	QuestionCount   int
	WrongGuessCount int
	Target          *string
}

// GameCompletionRecord: 게임 전체 완료 기록 구조체
type GameCompletionRecord struct {
	SessionID          string
	ChatID             string
	Category           string
	Result             GameResult
	Players            []PlayerCompletionRecord
	TotalQuestionCount int
	HintCount          int
	CompletedAt        time.Time
}

// StatsRecorder: 게임 통계를 비동기 또는 동기로 기록하는 레코더
type StatsRecorder struct {
	repo   *qrepo.Repository
	logger *slog.Logger

	// 비동기 처리용 (분석용 로그만)
	completionQueue    chan asyncRecord
	wg                 sync.WaitGroup
	stopOnce           sync.Once
	stopped            chan struct{}
	dropLogOnQueueFull bool
}

// NewStatsRecorder: 새로운 StatsRecorder 인스턴스를 생성합니다.
func NewStatsRecorder(repo *qrepo.Repository, logger *slog.Logger, cfg qconfig.StatsConfig) *StatsRecorder {
	if repo == nil {
		return nil
	}

	queueSize := cfg.QueueSize
	if queueSize <= 0 {
		queueSize = 100
	}
	workerCount := cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = 2
	}

	r := &StatsRecorder{
		repo:               repo,
		logger:             logger,
		completionQueue:    make(chan asyncRecord, queueSize),
		stopped:            make(chan struct{}),
		dropLogOnQueueFull: cfg.DropLogOnQueueFull,
	}

	// 백그라운드 워커 시작
	for i := 0; i < workerCount; i++ {
		r.wg.Add(1)
		go r.worker(i)
	}

	logger.Info("stats_recorder_started", "workers", workerCount, "queue_size", queueSize, "drop_on_full", cfg.DropLogOnQueueFull)
	return r
}

// Shutdown 정상 종료 - 대기 중인 작업 완료 후 종료
func (r *StatsRecorder) Shutdown() {
	if r == nil {
		return
	}

	r.stopOnce.Do(func() {
		close(r.stopped)
		close(r.completionQueue)
		r.wg.Wait()
		r.logger.Info("stats_recorder_shutdown_complete")
	})
}

func (r *StatsRecorder) worker(id int) {
	defer r.wg.Done()

	for ar := range r.completionQueue {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		r.processNonCriticalAsync(ctx, ar.record, ar.now)
		cancel()
	}

	r.logger.Debug("stats_worker_stopped", "worker_id", id)
}

// RecordGameStart: 게임 시작 정보를 기록합니다.
func (r *StatsRecorder) RecordGameStart(ctx context.Context, chatID string, userID string) {
	if r == nil || r.repo == nil {
		return
	}

	now := time.Now()
	if err := r.repo.RecordGameStart(ctx, chatID, userID, now); err != nil {
		r.logger.Warn("stats_game_start_failed", "chat_id", chatID, "user_id", userID, "err", err)
	}
}

// RecordGameCompletion 게임 완료 기록 (하이브리드 동기/비동기)
// - user_stats (사용자에게 표시되는 통계): 동기 처리 → Read-Your-Writes 일관성 보장
// - game_session, game_log (분석용 로그): 비동기 처리 → 응답 지연 최소화
func (r *StatsRecorder) RecordGameCompletion(ctx context.Context, record GameCompletionRecord) {
	if r == nil || r.repo == nil {
		return
	}

	record.ChatID = strings.TrimSpace(record.ChatID)
	record.Category = strings.TrimSpace(record.Category)
	record.SessionID = strings.TrimSpace(record.SessionID)

	if record.ChatID == "" || record.Category == "" || record.Result == "" {
		return
	}

	now := time.Now()

	// [동기] 사용자에게 표시되는 핵심 통계 먼저 처리
	r.processCriticalSync(ctx, record, now)

	// [비동기] 분석용 로그는 큐에 추가
	select {
	case r.completionQueue <- asyncRecord{record: record, now: now}:
		// 성공적으로 큐에 추가됨
	case <-r.stopped:
		r.logger.Warn("stats_recorder_stopped_dropping_record", "chat_id", record.ChatID)
	default:
		// 큐가 가득 참
		if r.dropLogOnQueueFull {
			// 드랍 옵션이 켜져있으면 경고 로그만 남기고 무시
			r.logger.Warn("stats_queue_full_dropped", "chat_id", record.ChatID)
		} else {
			// 기본: 동기로 처리 (fallback)
			r.logger.Warn("stats_queue_full_sync_fallback", "chat_id", record.ChatID)
			r.processNonCriticalAsync(ctx, record, now)
		}
	}
}

// asyncRecord 비동기 처리용 레코드
type asyncRecord struct {
	record GameCompletionRecord
	now    time.Time
}

// RecordGameCompletionSync 게임 완료 기록을 동기로 처리 (테스트용)
func (r *StatsRecorder) RecordGameCompletionSync(ctx context.Context, record GameCompletionRecord) {
	if r == nil || r.repo == nil {
		return
	}

	record.ChatID = strings.TrimSpace(record.ChatID)
	record.Category = strings.TrimSpace(record.Category)
	record.SessionID = strings.TrimSpace(record.SessionID)

	if record.ChatID == "" || record.Category == "" || record.Result == "" {
		return
	}

	now := time.Now()
	r.processCriticalSync(ctx, record, now)
	r.processNonCriticalAsync(ctx, record, now)
}

// processCriticalSync 사용자에게 표시되는 핵심 통계 동기 처리
// - user_stats: 게임 수, 승수, 베스트 스코어, 카테고리별 통계
// - nickname_map: 닉네임 조회용 (배치 처리로 DB 호출 최소화)
func (r *StatsRecorder) processCriticalSync(ctx context.Context, record GameCompletionRecord, now time.Time) {
	// 배치 닉네임 UPSERT (N개의 DB 호출 → 1개)
	nicknameEntries := make([]qrepo.NicknameEntry, 0, len(record.Players))
	for _, p := range record.Players {
		userID := strings.TrimSpace(p.UserID)
		if userID == "" {
			continue
		}
		nicknameEntries = append(nicknameEntries, qrepo.NicknameEntry{
			UserID:     userID,
			LastSender: p.Sender,
		})
	}

	if len(nicknameEntries) > 0 {
		if err := r.repo.BatchUpsertNicknames(ctx, record.ChatID, nicknameEntries, now); err != nil {
			r.logger.Warn("stats_batch_nickname_upsert_failed", "chat_id", record.ChatID, "count", len(nicknameEntries), "err", err)
		}
	}

	// 각 플레이어별 통계 업데이트 - 병렬 처리로 레이턴시 개선
	// 각 트랜잭션은 독립적인 user_stats 레코드를 다루므로 동시 실행 가능
	var failedCount atomic.Int32
	g, gctx := errgroup.WithContext(ctx)

	for _, p := range record.Players {
		p := p // 캡처용
		userID := strings.TrimSpace(p.UserID)
		if userID == "" {
			continue
		}

		g.Go(func() error {
			if err := r.repo.RecordGameCompletion(gctx, qrepo.GameCompletionParams{
				ChatID:                 record.ChatID,
				UserID:                 userID,
				Category:               record.Category,
				Result:                 qrepo.GameResult(record.Result),
				QuestionCount:          p.QuestionCount,
				HintCount:              record.HintCount,
				WrongGuessCount:        p.WrongGuessCount,
				Target:                 p.Target,
				TotalGameQuestionCount: record.TotalQuestionCount,
				CompletedAt:            record.CompletedAt,
				Now:                    now,
			}); err != nil {
				failedCount.Add(1)
				r.logger.Warn("stats_user_stats_record_failed", "chat_id", record.ChatID, "user_id", userID, "err", err)
			}
			// 개별 실패는 무시하고 다른 플레이어 처리 계속
			return nil
		})
	}

	// 모든 병렬 작업 완료 대기
	_ = g.Wait()

	if failed := failedCount.Load(); failed > 0 {
		r.logger.Warn("stats_completion_partial_failure", "chat_id", record.ChatID, "failed", failed, "total", len(record.Players))
	}
}

// processNonCriticalAsync 분석용 로그 처리 (비동기 또는 fallback 동기)
// - game_session: 게임 세션 메타데이터
// - game_log: 플레이어별 상세 기록
func (r *StatsRecorder) processNonCriticalAsync(ctx context.Context, record GameCompletionRecord, now time.Time) {
	participantCount := len(record.Players)
	if participantCount < 1 {
		participantCount = 1
	}

	// 게임 세션 기록
	if err := r.repo.RecordGameSession(ctx, qrepo.GameSessionParams{
		SessionID:        record.SessionID,
		ChatID:           record.ChatID,
		Category:         record.Category,
		Result:           qrepo.GameResult(record.Result),
		ParticipantCount: participantCount,
		QuestionCount:    record.TotalQuestionCount,
		HintCount:        record.HintCount,
		CompletedAt:      record.CompletedAt,
		Now:              now,
	}); err != nil {
		r.logger.Warn("stats_game_session_record_failed", "chat_id", record.ChatID, "err", err)
	}

	// 플레이어별 게임 로그
	for _, p := range record.Players {
		userID := strings.TrimSpace(p.UserID)
		if userID == "" {
			continue
		}

		if err := r.repo.RecordGameLog(ctx, qrepo.GameLogParams{
			ChatID:          record.ChatID,
			UserID:          userID,
			Sender:          p.Sender,
			Category:        record.Category,
			QuestionCount:   p.QuestionCount,
			HintCount:       record.HintCount,
			WrongGuessCount: p.WrongGuessCount,
			Result:          qrepo.GameResult(record.Result),
			Target:          p.Target,
			CompletedAt:     record.CompletedAt,
			Now:             now,
		}); err != nil {
			r.logger.Warn("stats_game_log_record_failed", "chat_id", record.ChatID, "user_id", userID, "err", err)
		}
	}
}
