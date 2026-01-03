import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { twentyQApi, turtleSoupApi } from '@/api/gameBots'
import { gameBotKeys } from '@/api/queryKeys'
import type {
    HintInjectRequest,
    SessionCleanupRequest,
    TurtleSoupPuzzleCreateRequest,
    TurtleSoupPuzzleUpdateRequest,
} from '@/types/gameBots'

export function useTwentyQStats() {
    return useQuery({
        queryKey: gameBotKeys.twentyQ.stats(),
        queryFn: twentyQApi.getStats,
        refetchInterval: 30000,
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
        refetchInterval: 10000,
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
