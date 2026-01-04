/**
 * Query Key Factory
 * TanStack Query의 queryKey를 중앙 관리하여 일관성 및 타입 안전성 확보
 *
 * @see https://tkdodo.eu/blog/effective-react-query-keys#use-query-key-factories
 */

import type { TraceSearchParams, MetricsParams } from '@/types'

export const queryKeys = {
    /** 멤버 관련 쿼리 키 */
    members: {
        all: ['members'] as const,
        detail: (id: number) => ['members', id] as const,
    },

    /** 알람 관련 쿼리 키 */
    alarms: {
        all: ['alarms'] as const,
    },

    /** 채팅방 관련 쿼리 키 */
    rooms: {
        all: ['rooms'] as const,
    },

    /** 통계 관련 쿼리 키 */
    stats: {
        summary: ['stats'] as const,
        channels: ['stats', 'channels'] as const,
    },

    /** 스트림 관련 쿼리 키 */
    streams: {
        live: ['streams', 'live'] as const,
        upcoming: ['streams', 'upcoming'] as const,
    },

    /** 로그 관련 쿼리 키 */
    logs: {
        all: ['logs'] as const,
        system: (file: string) => ['logs', 'system', file] as const,
        systemFiles: ['logs', 'system', 'files'] as const,
    },

    /** 설정 관련 쿼리 키 */
    settings: {
        all: ['settings'] as const,
    },

    /** Docker 관련 쿼리 키 */
    docker: {
        health: ['docker-health'] as const,
        containers: ['docker-containers'] as const,
    },

    /** 마일스톤 관련 쿼리 키 */
    milestones: {
        all: ['milestones'] as const,
        near: ['milestones', 'near'] as const,
        stats: ['milestones', 'stats'] as const,
    },

    /** Traces 관련 쿼리 키 */
    traces: {
        health: ['traces', 'health'] as const,
        services: ['traces', 'services'] as const,
        operations: (service: string) => ['traces', 'operations', service] as const,
        search: (params: TraceSearchParams) => ['traces', 'search', params] as const,
        detail: (traceId: string) => ['traces', 'detail', traceId] as const,
        dependencies: (lookback: string) => ['traces', 'dependencies', lookback] as const,
        metrics: (service: string, params?: MetricsParams) => ['traces', 'metrics', service, params] as const,
    },
    /** 시스템 상태 통합 쿼리 키 */
    status: {
        all: ['status'] as const,
        aggregated: ['status', 'aggregated'] as const,
    },
} as const

/** Game Bot 관련 쿼리 키 (TwentyQ, TurtleSoup) */
export const gameBotKeys = {
    /** 스무고개(TwentyQ) 관련 쿼리 키 */
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
    /** 거북이수프(TurtleSoup) 관련 쿼리 키 */
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

/** 타입 추출용 */
export type QueryKeys = typeof queryKeys
