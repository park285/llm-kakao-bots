# Game Bot Admin UI 프론트엔드 구현 가이드

## 개요

이 가이드는 TwentyQ 및 TurtleSoup 게임 봇의 Admin API를 프론트엔드(React/TypeScript)에서 구현하기 위한 완전한 참조 문서입니다.

**Base URL**: `/admin/api/{twentyq,turtle}/admin`

---

## 1. 타입 정의

`src/types/gameBots.ts` 파일에 추가:

```typescript
// ─────────────────────────────────────────────────────────────
// Common Types
// ─────────────────────────────────────────────────────────────

export interface PaginatedResponse<T> {
  status: string
  total: number
  limit: number
  offset: number
  data?: T
}

// ─────────────────────────────────────────────────────────────
// TwentyQ Types
// ─────────────────────────────────────────────────────────────

// 통합 통계
export interface TwentyQStats {
  totalPlayed: number
  totalCorrect: number
  totalSurrender: number
  successRate: number
  activeSessions: number
  totalParticipants: number
  last24HoursGames: number
}

export interface TwentyQStatsResponse {
  status: string
  stats: TwentyQStats
  durationMs: number
}

// 활성 세션
export interface TwentyQActiveSession {
  chatId: string
  category: string
  target: string
  ttlSeconds: number
}

export interface TwentyQActiveSessionsResponse {
  status: string
  sessions: TwentyQActiveSession[]
  count: number
}

// 세션 상세 (진행 중인 게임)
export interface QuestionHistoryItem {
  questionNumber: number
  question: string
  answer: string
  isChain: boolean
  userId?: string
}

export interface PlayerInfo {
  userId: string
  sender: string
}

export interface TwentyQSessionDetail {
  chatId: string
  target: string
  category: string
  intro: string
  questionCount: number
  hintCount: number
  ttlSeconds: number
}

export interface TwentyQSessionDetailResponse {
  status: string
  session: TwentyQSessionDetail
  history: QuestionHistoryItem[]
  players: PlayerInfo[]
}

// 게임 기록
export interface TwentyQGameRecord {
  sessionId: string
  chatId: string
  category: string
  target: string
  result: 'correct' | 'surrendered' | 'timeout'
  participantCount: number
  questionCount: number
  hintCount: number
  completedAt: string
}

export interface TwentyQGamesResponse {
  status: string
  games: TwentyQGameRecord[]
  total: number
  limit: number
  offset: number
}

// 게임 상세 (종료된 게임)
export interface TwentyQGameLog {
  id: number
  chatId: string
  userId: string
  sender: string
  category: string
  questionCount: number
  hintCount: number
  wrongGuessCount: number
  result: string
  target?: string
  completedAt: string
}

export interface TwentyQAuditLog {
  id: number
  sessionId: string
  questionIndex: number
  verdict: 'AI_CORRECT' | 'AI_WRONG' | 'UNCLEAR'
  reason: string
  adminUserId: string
  createdAt: string
}

export interface TwentyQRefundLog {
  id: number
  sessionId: string
  userId: string
  adminUserId: string
  reason: string
  createdAt: string
}

export interface TwentyQGameDetailResponse {
  status: string
  session: {
    sessionId: string
    chatId: string
    category: string
    target: string
    result: string
    participantCount: number
    questionCount: number
    hintCount: number
    completedAt: string
  }
  logs: TwentyQGameLog[]
  audits: TwentyQAuditLog[]
  refunds: TwentyQRefundLog[]
}

// 리더보드
export interface TwentyQLeaderboardEntry {
  rank: number
  chatId: string
  chatName: string
  userId: string
  userName: string
  bestScore: number
  bestCategory: string
  bestTarget: string
  bestAt: string
}

export interface TwentyQLeaderboardResponse {
  status: string
  leaderboard: TwentyQLeaderboardEntry[]
  count: number
}

// 동의어
export interface TwentyQSynonym {
  canonical: string
  aliases: string[]
}

export interface TwentyQSynonymsResponse {
  status: string
  synonyms: TwentyQSynonym[]
  count: number
}

// 카테고리 통계
export interface TwentyQCategoryStats {
  category: string
  totalGames: number
  successCount: number
  surrenderRate: number
}

export interface TwentyQCategoryStatsResponse {
  status: string
  categories: TwentyQCategoryStats[]
  count: number
}

// 닉네임 매핑
export interface TwentyQNicknameMap {
  userId: string
  lastSender: string
  lastSeenAt: string
}

export interface TwentyQNicknamesResponse {
  status: string
  nicknames: TwentyQNicknameMap[]
  count: number
}

// 유저 통계
export interface TwentyQUserStats {
  id: string
  chatId: string
  userId: string
  totalGamesStarted: number
  totalGamesCompleted: number
  totalSurrenders: number
  totalQuestionsAsked: number
  totalHintsUsed: number
  totalWrongGuesses: number
  bestScoreQuestionCnt?: number
  bestScoreWrongGuess?: number
  bestScoreTarget?: string
  bestScoreCategory?: string
  bestScoreAchievedAt?: string
  categoryStatsJson?: string
  createdAt: string
  updatedAt: string
}

export interface TwentyQUserStatsListResponse {
  status: string
  stats: TwentyQUserStats[]
  total: number
  limit: number
  offset: number
}

// 오디트/리펀드 로그 조회
export interface TwentyQAuditLogsResponse {
  status: string
  logs: TwentyQAuditLog[]
  total: number
  limit: number
  offset: number
}

export interface TwentyQRefundLogsResponse {
  status: string
  logs: TwentyQRefundLog[]
  total: number
  limit: number
  offset: number
}

// ─────────────────────────────────────────────────────────────
// TurtleSoup Types
// ─────────────────────────────────────────────────────────────

// 통합 통계
export interface TurtleSoupStats {
  activeSessions: number
  totalSolved: number
  totalFailed: number
  solveRate: number
  avgQuestions: number
  avgHintsPerGame: number
  last24HoursSolve: number
}

export interface TurtleSoupStatsResponse {
  status: string
  stats: TurtleSoupStats
}

// 활성 세션
export interface TurtleSoupActiveSession {
  sessionId: string
  chatId: string
  questionCount: number
  hintCount: number
  ttlSeconds: number
}

export interface TurtleSoupActiveSessionsResponse {
  status: string
  sessions: TurtleSoupActiveSession[]
  count: number
}

// 퍼즐 CMS
export interface TurtleSoupPuzzle {
  id: number
  title: string
  scenario: string
  solution: string
  category: string
  difficulty: number
  hintsJson: string
  status: 'draft' | 'test' | 'published'
  authorId: string
  playCount: number
  solveCount: number
  avgQuestion: number
  createdAt: string
  updatedAt: string
}

export interface TurtleSoupPuzzlesResponse {
  status: string
  puzzles: TurtleSoupPuzzle[]
  total: number
  limit: number
  offset: number
}

export interface TurtleSoupPuzzleCreateRequest {
  title: string
  scenario: string
  solution: string
  category?: string
  difficulty?: number
  hints?: string[]
  authorId?: string
}

export interface TurtleSoupPuzzleUpdateRequest {
  title?: string
  scenario?: string
  solution?: string
  category?: string
  difficulty?: number
  status?: 'draft' | 'test' | 'published'
  hints?: string[]
}

// 퍼즐 통계
export interface TurtleSoupPuzzleStats {
  totalPuzzles: number
  publishedCount: number
  draftCount: number
  totalPlays: number
  totalSolves: number
  overallSolveRate: number
}

export interface TurtleSoupCategoryStats {
  category: string
  totalGames: number
  solveCount: number
  solveRate: number
}

export interface TurtleSoupPuzzleStatsResponse {
  status: string
  stats: TurtleSoupPuzzleStats
  categoryStats: TurtleSoupCategoryStats[]
}

// 게임 아카이브
export interface TurtleSoupGameArchive {
  id: number
  sessionId: string
  chatId: string
  puzzleId?: number
  questionCount: number
  hintsUsed: number
  result: 'solved' | 'surrendered' | 'timeout'
  historyJson: string
  startedAt: string
  completedAt: string
  createdAt: string
}

export interface TurtleSoupArchivesResponse {
  status: string
  archives: TurtleSoupGameArchive[]
  total: number
  limit: number
  offset: number
}

// 힌트 주입 요청
export interface HintInjectRequest {
  message: string
  asBot?: boolean
}

// 세션 정리 요청
export interface SessionCleanupRequest {
  olderThanHours: number
}

export interface SessionCleanupResponse {
  status: string
  deletedCount: number
}
```

---

## 2. API 클라이언트

`src/api/gameBots.ts` 파일 생성:

```typescript
import apiClient from '@/api/client'
import type {
  // TwentyQ Types
  TwentyQStatsResponse,
  TwentyQActiveSessionsResponse,
  TwentyQSessionDetailResponse,
  TwentyQGamesResponse,
  TwentyQGameDetailResponse,
  TwentyQLeaderboardResponse,
  TwentyQSynonymsResponse,
  TwentyQCategoryStatsResponse,
  TwentyQNicknamesResponse,
  TwentyQUserStatsListResponse,
  TwentyQAuditLogsResponse,
  TwentyQRefundLogsResponse,
  // TurtleSoup Types
  TurtleSoupStatsResponse,
  TurtleSoupActiveSessionsResponse,
  TurtleSoupPuzzlesResponse,
  TurtleSoupPuzzle,
  TurtleSoupPuzzleCreateRequest,
  TurtleSoupPuzzleUpdateRequest,
  TurtleSoupPuzzleStatsResponse,
  TurtleSoupArchivesResponse,
  // Common
  HintInjectRequest,
  SessionCleanupRequest,
  SessionCleanupResponse,
} from '@/types/gameBots'
import type { ApiResponse } from '@/types'

// ─────────────────────────────────────────────────────────────
// TwentyQ API
// ─────────────────────────────────────────────────────────────

const TWENTYQ_BASE = '/twentyq/admin'

export const twentyQApi = {
  // 통계
  getStats: async (): Promise<TwentyQStatsResponse> => {
    const response = await apiClient.get<TwentyQStatsResponse>(`${TWENTYQ_BASE}/stats`)
    return response.data
  },

  getCategoryStats: async (): Promise<TwentyQCategoryStatsResponse> => {
    const response = await apiClient.get<TwentyQCategoryStatsResponse>(`${TWENTYQ_BASE}/stats/categories`)
    return response.data
  },

  // 세션 관리
  getSessions: async (): Promise<TwentyQActiveSessionsResponse> => {
    const response = await apiClient.get<TwentyQActiveSessionsResponse>(`${TWENTYQ_BASE}/sessions`)
    return response.data
  },

  getSessionDetail: async (chatId: string): Promise<TwentyQSessionDetailResponse> => {
    const response = await apiClient.get<TwentyQSessionDetailResponse>(
      `${TWENTYQ_BASE}/sessions/${encodeURIComponent(chatId)}`
    )
    return response.data
  },

  deleteSession: async (chatId: string): Promise<ApiResponse> => {
    const response = await apiClient.delete<ApiResponse>(
      `${TWENTYQ_BASE}/sessions/${encodeURIComponent(chatId)}`
    )
    return response.data
  },

  injectHint: async (chatId: string, request: HintInjectRequest): Promise<ApiResponse> => {
    const response = await apiClient.post<ApiResponse>(
      `${TWENTYQ_BASE}/sessions/${encodeURIComponent(chatId)}/hint`,
      request
    )
    return response.data
  },

  cleanupSessions: async (request: SessionCleanupRequest): Promise<SessionCleanupResponse> => {
    const response = await apiClient.post<SessionCleanupResponse>(
      `${TWENTYQ_BASE}/sessions/cleanup`,
      request
    )
    return response.data
  },

  // 게임 기록
  getGames: async (params?: { 
    chatId?: string
    category?: string
    result?: string
    limit?: number
    offset?: number 
  }): Promise<TwentyQGamesResponse> => {
    const response = await apiClient.get<TwentyQGamesResponse>(`${TWENTYQ_BASE}/games`, { params })
    return response.data
  },

  getGameDetail: async (sessionId: string): Promise<TwentyQGameDetailResponse> => {
    const response = await apiClient.get<TwentyQGameDetailResponse>(
      `${TWENTYQ_BASE}/games/${encodeURIComponent(sessionId)}`
    )
    return response.data
  },

  // 리더보드
  getLeaderboard: async (params?: { 
    chatId?: string
    limit?: number 
  }): Promise<TwentyQLeaderboardResponse> => {
    const response = await apiClient.get<TwentyQLeaderboardResponse>(
      `${TWENTYQ_BASE}/leaderboard`, 
      { params }
    )
    return response.data
  },

  // 동의어 CMS
  getSynonyms: async (query?: string): Promise<TwentyQSynonymsResponse> => {
    const response = await apiClient.get<TwentyQSynonymsResponse>(
      `${TWENTYQ_BASE}/synonyms`,
      { params: query ? { query } : undefined }
    )
    return response.data
  },

  createSynonym: async (alias: string, canonical: string): Promise<ApiResponse> => {
    const response = await apiClient.post<ApiResponse>(`${TWENTYQ_BASE}/synonyms`, {
      alias,
      canonical,
    })
    return response.data
  },

  deleteSynonym: async (alias: string): Promise<ApiResponse> => {
    const response = await apiClient.delete<ApiResponse>(
      `${TWENTYQ_BASE}/synonyms/${encodeURIComponent(alias)}`
    )
    return response.data
  },

  // 닉네임 매핑
  getNicknames: async (params?: { 
    chatId?: string
    limit?: number 
  }): Promise<TwentyQNicknamesResponse> => {
    const response = await apiClient.get<TwentyQNicknamesResponse>(
      `${TWENTYQ_BASE}/nicknames`,
      { params }
    )
    return response.data
  },

  // 유저 통계
  getUserStatsList: async (params?: { 
    chatId?: string
    limit?: number
    offset?: number 
  }): Promise<TwentyQUserStatsListResponse> => {
    const response = await apiClient.get<TwentyQUserStatsListResponse>(
      `${TWENTYQ_BASE}/users/stats`,
      { params }
    )
    return response.data
  },

  getUserStats: async (userId: string, chatId?: string): Promise<{ status: string; stats: unknown[] }> => {
    const response = await apiClient.get<{ status: string; stats: unknown[] }>(
      `${TWENTYQ_BASE}/users/${encodeURIComponent(userId)}/stats`,
      { params: chatId ? { chatId } : undefined }
    )
    return response.data
  },

  resetUserStats: async (userId: string, chatId?: string): Promise<ApiResponse & { deletedCount: number }> => {
    const response = await apiClient.delete<ApiResponse & { deletedCount: number }>(
      `${TWENTYQ_BASE}/users/${encodeURIComponent(userId)}/stats`,
      { params: chatId ? { chatId } : undefined }
    )
    return response.data
  },

  // 오디트/리뷰
  createAudit: async (sessionId: string, data: {
    questionIndex: number
    verdict: 'AI_CORRECT' | 'AI_WRONG' | 'UNCLEAR'
    reason: string
    adminUserId: string
  }): Promise<ApiResponse> => {
    const response = await apiClient.post<ApiResponse>(
      `${TWENTYQ_BASE}/games/${encodeURIComponent(sessionId)}/audit`,
      data
    )
    return response.data
  },

  createRefund: async (sessionId: string, data: {
    userId: string
    adminUserId: string
    reason: string
  }): Promise<ApiResponse> => {
    const response = await apiClient.post<ApiResponse>(
      `${TWENTYQ_BASE}/games/${encodeURIComponent(sessionId)}/refund`,
      data
    )
    return response.data
  },

  // 로그 조회
  getAuditLogs: async (params?: { 
    sessionId?: string
    limit?: number
    offset?: number 
  }): Promise<TwentyQAuditLogsResponse> => {
    const response = await apiClient.get<TwentyQAuditLogsResponse>(
      `${TWENTYQ_BASE}/audits`,
      { params }
    )
    return response.data
  },

  getRefundLogs: async (params?: { 
    sessionId?: string
    userId?: string
    limit?: number
    offset?: number 
  }): Promise<TwentyQRefundLogsResponse> => {
    const response = await apiClient.get<TwentyQRefundLogsResponse>(
      `${TWENTYQ_BASE}/refunds`,
      { params }
    )
    return response.data
  },
}

// ─────────────────────────────────────────────────────────────
// TurtleSoup API
// ─────────────────────────────────────────────────────────────

const TURTLE_BASE = '/turtle/admin'

export const turtleSoupApi = {
  // 통계
  getStats: async (): Promise<TurtleSoupStatsResponse> => {
    const response = await apiClient.get<TurtleSoupStatsResponse>(`${TURTLE_BASE}/stats`)
    return response.data
  },

  // 세션 관리
  getSessions: async (): Promise<TurtleSoupActiveSessionsResponse> => {
    const response = await apiClient.get<TurtleSoupActiveSessionsResponse>(`${TURTLE_BASE}/sessions`)
    return response.data
  },

  deleteSession: async (sessionId: string): Promise<ApiResponse> => {
    const response = await apiClient.delete<ApiResponse>(
      `${TURTLE_BASE}/sessions/${encodeURIComponent(sessionId)}`
    )
    return response.data
  },

  injectHint: async (sessionId: string, request: HintInjectRequest): Promise<ApiResponse> => {
    const response = await apiClient.post<ApiResponse>(
      `${TURTLE_BASE}/sessions/${encodeURIComponent(sessionId)}/inject`,
      request
    )
    return response.data
  },

  cleanupSessions: async (request: SessionCleanupRequest): Promise<SessionCleanupResponse> => {
    const response = await apiClient.post<SessionCleanupResponse>(
      `${TURTLE_BASE}/sessions/cleanup`,
      request
    )
    return response.data
  },

  // 퍼즐 CMS
  getPuzzles: async (params?: { 
    status?: string
    limit?: number
    offset?: number 
  }): Promise<TurtleSoupPuzzlesResponse> => {
    const response = await apiClient.get<TurtleSoupPuzzlesResponse>(
      `${TURTLE_BASE}/puzzles`,
      { params }
    )
    return response.data
  },

  getPuzzle: async (id: number): Promise<{ status: string; puzzle: TurtleSoupPuzzle }> => {
    const response = await apiClient.get<{ status: string; puzzle: TurtleSoupPuzzle }>(
      `${TURTLE_BASE}/puzzles/${String(id)}`
    )
    return response.data
  },

  createPuzzle: async (data: TurtleSoupPuzzleCreateRequest): Promise<{ status: string; puzzle: TurtleSoupPuzzle }> => {
    const response = await apiClient.post<{ status: string; puzzle: TurtleSoupPuzzle }>(
      `${TURTLE_BASE}/puzzles`,
      data
    )
    return response.data
  },

  updatePuzzle: async (id: number, data: TurtleSoupPuzzleUpdateRequest): Promise<{ status: string; puzzle: TurtleSoupPuzzle }> => {
    const response = await apiClient.put<{ status: string; puzzle: TurtleSoupPuzzle }>(
      `${TURTLE_BASE}/puzzles/${String(id)}`,
      data
    )
    return response.data
  },

  deletePuzzle: async (id: number): Promise<ApiResponse> => {
    const response = await apiClient.delete<ApiResponse>(`${TURTLE_BASE}/puzzles/${String(id)}`)
    return response.data
  },

  getPuzzleStats: async (): Promise<TurtleSoupPuzzleStatsResponse> => {
    const response = await apiClient.get<TurtleSoupPuzzleStatsResponse>(`${TURTLE_BASE}/puzzles/stats`)
    return response.data
  },

  // 아카이브
  getArchives: async (params?: { 
    result?: string
    limit?: number
    offset?: number 
  }): Promise<TurtleSoupArchivesResponse> => {
    const response = await apiClient.get<TurtleSoupArchivesResponse>(
      `${TURTLE_BASE}/archives`,
      { params }
    )
    return response.data
  },
}
```

---

## 3. Query Keys

`src/api/queryKeys.ts`에 추가:

```typescript
// Game Bots Query Keys
export const gameBotKeys = {
  // TwentyQ
  twentyQ: {
    all: ['twentyq'] as const,
    stats: () => [...gameBotKeys.twentyQ.all, 'stats'] as const,
    categoryStats: () => [...gameBotKeys.twentyQ.all, 'categoryStats'] as const,
    sessions: () => [...gameBotKeys.twentyQ.all, 'sessions'] as const,
    sessionDetail: (chatId: string) => [...gameBotKeys.twentyQ.sessions(), chatId] as const,
    games: (filters?: object) => [...gameBotKeys.twentyQ.all, 'games', filters] as const,
    gameDetail: (sessionId: string) => [...gameBotKeys.twentyQ.all, 'game', sessionId] as const,
    leaderboard: (filters?: object) => [...gameBotKeys.twentyQ.all, 'leaderboard', filters] as const,
    synonyms: (query?: string) => [...gameBotKeys.twentyQ.all, 'synonyms', query] as const,
    nicknames: (filters?: object) => [...gameBotKeys.twentyQ.all, 'nicknames', filters] as const,
    userStats: (filters?: object) => [...gameBotKeys.twentyQ.all, 'userStats', filters] as const,
    auditLogs: (filters?: object) => [...gameBotKeys.twentyQ.all, 'auditLogs', filters] as const,
    refundLogs: (filters?: object) => [...gameBotKeys.twentyQ.all, 'refundLogs', filters] as const,
  },

  // TurtleSoup
  turtleSoup: {
    all: ['turtleSoup'] as const,
    stats: () => [...gameBotKeys.turtleSoup.all, 'stats'] as const,
    sessions: () => [...gameBotKeys.turtleSoup.all, 'sessions'] as const,
    puzzles: (filters?: object) => [...gameBotKeys.turtleSoup.all, 'puzzles', filters] as const,
    puzzle: (id: number) => [...gameBotKeys.turtleSoup.all, 'puzzle', id] as const,
    puzzleStats: () => [...gameBotKeys.turtleSoup.all, 'puzzleStats'] as const,
    archives: (filters?: object) => [...gameBotKeys.turtleSoup.all, 'archives', filters] as const,
  },
}
```

---

## 4. React Query Hooks

`src/hooks/useGameBots.ts` 파일 생성:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { twentyQApi, turtleSoupApi } from '@/api/gameBots'
import { gameBotKeys } from '@/api/queryKeys'
import type {
  HintInjectRequest,
  SessionCleanupRequest,
  TurtleSoupPuzzleCreateRequest,
  TurtleSoupPuzzleUpdateRequest,
} from '@/types/gameBots'

// ─────────────────────────────────────────────────────────────
// TwentyQ Hooks
// ─────────────────────────────────────────────────────────────

export function useTwentyQStats() {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.stats(),
    queryFn: twentyQApi.getStats,
    refetchInterval: 30000, // 30초마다 갱신
  })
}

export function useTwentyQCategoryStats() {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.categoryStats(),
    queryFn: twentyQApi.getCategoryStats,
  })
}

export function useTwentyQSessions() {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.sessions(),
    queryFn: twentyQApi.getSessions,
    refetchInterval: 10000, // 10초마다 갱신
  })
}

export function useTwentyQSessionDetail(chatId: string) {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.sessionDetail(chatId),
    queryFn: () => twentyQApi.getSessionDetail(chatId),
    enabled: !!chatId,
  })
}

export function useTwentyQGames(params?: Parameters<typeof twentyQApi.getGames>[0]) {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.games(params),
    queryFn: () => twentyQApi.getGames(params),
  })
}

export function useTwentyQGameDetail(sessionId: string) {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.gameDetail(sessionId),
    queryFn: () => twentyQApi.getGameDetail(sessionId),
    enabled: !!sessionId,
  })
}

export function useTwentyQLeaderboard(params?: Parameters<typeof twentyQApi.getLeaderboard>[0]) {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.leaderboard(params),
    queryFn: () => twentyQApi.getLeaderboard(params),
  })
}

export function useTwentyQSynonyms(query?: string) {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.synonyms(query),
    queryFn: () => twentyQApi.getSynonyms(query),
  })
}

export function useTwentyQUserStats(params?: Parameters<typeof twentyQApi.getUserStatsList>[0]) {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.userStats(params),
    queryFn: () => twentyQApi.getUserStatsList(params),
  })
}

export function useTwentyQAuditLogs(params?: Parameters<typeof twentyQApi.getAuditLogs>[0]) {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.auditLogs(params),
    queryFn: () => twentyQApi.getAuditLogs(params),
  })
}

export function useTwentyQRefundLogs(params?: Parameters<typeof twentyQApi.getRefundLogs>[0]) {
  return useQuery({
    queryKey: gameBotKeys.twentyQ.refundLogs(params),
    queryFn: () => twentyQApi.getRefundLogs(params),
  })
}

// Mutations
export function useTwentyQDeleteSession() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (chatId: string) => twentyQApi.deleteSession(chatId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.twentyQ.sessions() })
    },
  })
}

export function useTwentyQInjectHint() {
  return useMutation({
    mutationFn: ({ chatId, request }: { chatId: string; request: HintInjectRequest }) =>
      twentyQApi.injectHint(chatId, request),
  })
}

export function useTwentyQCleanupSessions() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: SessionCleanupRequest) => twentyQApi.cleanupSessions(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.twentyQ.sessions() })
    },
  })
}

export function useTwentyQCreateSynonym() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ alias, canonical }: { alias: string; canonical: string }) =>
      twentyQApi.createSynonym(alias, canonical),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.twentyQ.synonyms() })
    },
  })
}

export function useTwentyQDeleteSynonym() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (alias: string) => twentyQApi.deleteSynonym(alias),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.twentyQ.synonyms() })
    },
  })
}

export function useTwentyQResetUserStats() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, chatId }: { userId: string; chatId?: string }) =>
      twentyQApi.resetUserStats(userId, chatId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.twentyQ.userStats() })
    },
  })
}

// ─────────────────────────────────────────────────────────────
// TurtleSoup Hooks
// ─────────────────────────────────────────────────────────────

export function useTurtleSoupStats() {
  return useQuery({
    queryKey: gameBotKeys.turtleSoup.stats(),
    queryFn: turtleSoupApi.getStats,
    refetchInterval: 30000,
  })
}

export function useTurtleSoupSessions() {
  return useQuery({
    queryKey: gameBotKeys.turtleSoup.sessions(),
    queryFn: turtleSoupApi.getSessions,
    refetchInterval: 10000,
  })
}

export function useTurtleSoupPuzzles(params?: Parameters<typeof turtleSoupApi.getPuzzles>[0]) {
  return useQuery({
    queryKey: gameBotKeys.turtleSoup.puzzles(params),
    queryFn: () => turtleSoupApi.getPuzzles(params),
  })
}

export function useTurtleSoupPuzzle(id: number) {
  return useQuery({
    queryKey: gameBotKeys.turtleSoup.puzzle(id),
    queryFn: () => turtleSoupApi.getPuzzle(id),
    enabled: id > 0,
  })
}

export function useTurtleSoupPuzzleStats() {
  return useQuery({
    queryKey: gameBotKeys.turtleSoup.puzzleStats(),
    queryFn: turtleSoupApi.getPuzzleStats,
  })
}

export function useTurtleSoupArchives(params?: Parameters<typeof turtleSoupApi.getArchives>[0]) {
  return useQuery({
    queryKey: gameBotKeys.turtleSoup.archives(params),
    queryFn: () => turtleSoupApi.getArchives(params),
  })
}

// Mutations
export function useTurtleSoupDeleteSession() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (sessionId: string) => turtleSoupApi.deleteSession(sessionId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.turtleSoup.sessions() })
    },
  })
}

export function useTurtleSoupInjectHint() {
  return useMutation({
    mutationFn: ({ sessionId, request }: { sessionId: string; request: HintInjectRequest }) =>
      turtleSoupApi.injectHint(sessionId, request),
  })
}

export function useTurtleSoupCleanupSessions() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: SessionCleanupRequest) => turtleSoupApi.cleanupSessions(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.turtleSoup.sessions() })
    },
  })
}

export function useTurtleSoupCreatePuzzle() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: TurtleSoupPuzzleCreateRequest) => turtleSoupApi.createPuzzle(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.turtleSoup.puzzles() })
    },
  })
}

export function useTurtleSoupUpdatePuzzle() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: TurtleSoupPuzzleUpdateRequest }) =>
      turtleSoupApi.updatePuzzle(id, data),
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.turtleSoup.puzzle(variables.id) })
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.turtleSoup.puzzles() })
    },
  })
}

export function useTurtleSoupDeletePuzzle() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => turtleSoupApi.deletePuzzle(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: gameBotKeys.turtleSoup.puzzles() })
    },
  })
}
```

---

## 5. 컴포넌트 구조 제안

```
src/components/gameBots/
├── TwentyQ/
│   ├── TwentyQDashboard.tsx        # 통합 대시보드 (통계 카드)
│   ├── TwentyQSessionsTable.tsx    # 활성 세션 테이블
│   ├── TwentyQSessionDetail.tsx    # 세션 상세 + Q&A 히스토리
│   ├── TwentyQGamesTable.tsx       # 게임 기록 테이블
│   ├── TwentyQGameReplay.tsx       # 게임 리플레이 뷰어
│   ├── TwentyQLeaderboard.tsx      # 리더보드
│   ├── TwentyQSynonymManager.tsx   # 동의어 CMS
│   ├── TwentyQUserStatsTable.tsx   # 유저 통계 관리
│   └── TwentyQAuditPanel.tsx       # 오디트/리펀드 패널
│
├── TurtleSoup/
│   ├── TurtleSoupDashboard.tsx     # 통합 대시보드
│   ├── TurtleSoupSessionsTable.tsx # 활성 세션 테이블
│   ├── TurtleSoupPuzzleEditor.tsx  # 퍼즐 CMS 에디터
│   ├── TurtleSoupPuzzleList.tsx    # 퍼즐 목록
│   ├── TurtleSoupArchives.tsx      # 게임 아카이브
│   └── TurtleSoupStats.tsx         # 통계 차트
│
└── shared/
    ├── SessionCard.tsx             # 세션 카드 컴포넌트
    ├── HintInjectModal.tsx         # 힌트 주입 모달
    ├── CleanupDialog.tsx           # 세션 정리 다이얼로그
    └── GameStatsCard.tsx           # 통계 카드 컴포넌트
```

---

## 6. 라우팅 설정

`App.tsx`에 추가:

```tsx
import { Routes, Route } from 'react-router-dom'

// Game Bot Pages
import TwentyQPage from '@/pages/TwentyQPage'
import TurtleSoupPage from '@/pages/TurtleSoupPage'

// ...

<Routes>
  {/* 기존 라우트 */}
  
  {/* Game Bots */}
  <Route path="/games/twentyq/*" element={<TwentyQPage />} />
  <Route path="/games/turtlesoup/*" element={<TurtleSoupPage />} />
</Routes>
```

---

## 7. 네비게이션 추가

`AppLayout.tsx` 사이드바에 추가:

```tsx
const gameBotsNavItems = [
  {
    title: 'TwentyQ (스무고개)',
    icon: <Brain className="h-4 w-4" />,
    href: '/games/twentyq',
    subItems: [
      { title: '대시보드', href: '/games/twentyq' },
      { title: '활성 세션', href: '/games/twentyq/sessions' },
      { title: '게임 기록', href: '/games/twentyq/games' },
      { title: '리더보드', href: '/games/twentyq/leaderboard' },
      { title: '동의어 관리', href: '/games/twentyq/synonyms' },
      { title: '유저 통계', href: '/games/twentyq/users' },
    ],
  },
  {
    title: 'TurtleSoup (바다거북스프)',
    icon: <Puzzle className="h-4 w-4" />,
    href: '/games/turtlesoup',
    subItems: [
      { title: '대시보드', href: '/games/turtlesoup' },
      { title: '활성 세션', href: '/games/turtlesoup/sessions' },
      { title: '퍼즐 관리', href: '/games/turtlesoup/puzzles' },
      { title: '아카이브', href: '/games/turtlesoup/archives' },
    ],
  },
]
```

---

## 8. 주요 컴포넌트 예시

### TwentyQSessionDetail.tsx

```tsx
import { useTwentyQSessionDetail, useTwentyQInjectHint, useTwentyQDeleteSession } from '@/hooks/useGameBots'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useState } from 'react'

interface Props {
  chatId: string
}

export function TwentyQSessionDetail({ chatId }: Props) {
  const { data, isLoading } = useTwentyQSessionDetail(chatId)
  const injectHint = useTwentyQInjectHint()
  const deleteSession = useTwentyQDeleteSession()
  const [hintMessage, setHintMessage] = useState('')

  if (isLoading) return <div>로딩 중...</div>
  if (!data) return <div>세션을 찾을 수 없습니다</div>

  const { session, history, players } = data

  const handleInjectHint = async () => {
    if (!hintMessage.trim()) return
    await injectHint.mutateAsync({ chatId, request: { message: hintMessage } })
    setHintMessage('')
  }

  return (
    <div className="space-y-6">
      {/* 세션 정보 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>세션 정보</span>
            <Badge variant={session.ttlSeconds > 1800 ? 'default' : 'destructive'}>
              TTL: {Math.floor(session.ttlSeconds / 60)}분
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4">
          <div><strong>카테고리:</strong> {session.category}</div>
          <div><strong>정답:</strong> {session.target}</div>
          <div><strong>질문 수:</strong> {session.questionCount}</div>
          <div><strong>힌트 사용:</strong> {session.hintCount}</div>
        </CardContent>
      </Card>

      {/* Q&A 히스토리 */}
      <Card>
        <CardHeader><CardTitle>Q&A 히스토리</CardTitle></CardHeader>
        <CardContent>
          <div className="space-y-3">
            {history.map((item, idx) => (
              <div key={idx} className="border-l-2 pl-4 py-2">
                <div className="font-medium">Q{item.questionNumber}: {item.question}</div>
                <div className="text-muted-foreground">A: {item.answer}</div>
                {item.isChain && <Badge variant="outline">체인 질문</Badge>}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 참여자 목록 */}
      <Card>
        <CardHeader><CardTitle>참여자 ({players.length}명)</CardTitle></CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {players.map((p, idx) => (
              <Badge key={idx} variant="secondary">{p.sender}</Badge>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* GM 개입 */}
      <Card>
        <CardHeader><CardTitle>GM 개입</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <input
              type="text"
              value={hintMessage}
              onChange={(e) => setHintMessage(e.target.value)}
              placeholder="힌트 메시지 입력..."
              className="flex-1 px-3 py-2 border rounded"
            />
            <Button onClick={handleInjectHint} disabled={injectHint.isPending}>
              힌트 주입
            </Button>
          </div>
          <Button 
            variant="destructive" 
            onClick={() => deleteSession.mutate(chatId)}
            disabled={deleteSession.isPending}
          >
            세션 강제 종료
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
```

---

## 9. 변경 이력

| 날짜 | 버전 | 변경 내용 |
|:---|:---|:---|
| 2026-01-03 | 1.0.0 | 최초 작성 - TwentyQ/TurtleSoup 전체 API 커버 |
