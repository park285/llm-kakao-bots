
export interface PaginatedResponse<T> {
    status: string
    total: number
    limit: number
    offset: number
    data?: T
}

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

export interface TwentyQSynonym {
    canonical: string
    aliases: string[]
}

export interface TwentyQSynonymsResponse {
    status: string
    synonyms: TwentyQSynonym[]
    count: number
}

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

export interface HintInjectRequest {
    message: string
    asBot?: boolean
}

export interface SessionCleanupRequest {
    olderThanHours: number
}

export interface SessionCleanupResponse {
    status: string
    deletedCount: number
}
