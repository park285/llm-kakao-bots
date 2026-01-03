import apiClient from '@/api/client'
import type {
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
    TurtleSoupStatsResponse,
    TurtleSoupActiveSessionsResponse,
    TurtleSoupPuzzlesResponse,
    TurtleSoupPuzzle,
    TurtleSoupPuzzleCreateRequest,
    TurtleSoupPuzzleUpdateRequest,
    TurtleSoupPuzzleStatsResponse,
    TurtleSoupArchivesResponse,
    HintInjectRequest,
    SessionCleanupRequest,
    SessionCleanupResponse,
} from '@/types/gameBots'
import type { ApiResponse } from '@/types'

const TWENTYQ_BASE = '/twentyq/admin'

export const twentyQApi = {
    getStats: async (): Promise<TwentyQStatsResponse> => {
        const response = await apiClient.get<TwentyQStatsResponse>(`${TWENTYQ_BASE}/stats`)
        return response.data
    },

    getCategoryStats: async (): Promise<TwentyQCategoryStatsResponse> => {
        const response = await apiClient.get<TwentyQCategoryStatsResponse>(`${TWENTYQ_BASE}/stats/categories`)
        return response.data
    },

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

const TURTLE_BASE = '/turtle/admin'

export const turtleSoupApi = {
    getStats: async (): Promise<TurtleSoupStatsResponse> => {
        const response = await apiClient.get<TurtleSoupStatsResponse>(`${TURTLE_BASE}/stats`)
        return response.data
    },

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
