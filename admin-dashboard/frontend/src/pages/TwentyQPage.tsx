import { Routes, Route, Navigate } from 'react-router-dom'
import TwentyQDashboard from '@/components/gameBots/TwentyQ/TwentyQDashboard'
import TwentyQSessionsTable from '@/components/gameBots/TwentyQ/TwentyQSessionsTable'
// import TwentyQSessionDetail from '@/components/gameBots/TwentyQ/TwentyQSessionDetail'
// import TwentyQGamesTable from '@/components/gameBots/TwentyQ/TwentyQGamesTable'
// import TwentyQLeaderboard from '@/components/gameBots/TwentyQ/TwentyQLeaderboard'
// import TwentyQSynonymManager from '@/components/gameBots/TwentyQ/TwentyQSynonymManager'
// import TwentyQUserStatsTable from '@/components/gameBots/TwentyQ/TwentyQUserStatsTable'

export default function TwentyQPage() {
    return (
        <Routes>
            <Route index element={<TwentyQDashboard />} />
            <Route path="sessions" element={<TwentyQSessionsTable />} />
            {/* 
      <Route path="sessions/:chatId" element={<TwentyQSessionDetail />} />
      <Route path="games" element={<TwentyQGamesTable />} />
      <Route path="leaderboard" element={<TwentyQLeaderboard />} />
      <Route path="synonyms" element={<TwentyQSynonymManager />} />
      <Route path="users" element={<TwentyQUserStatsTable />} /> 
      */}
            <Route path="*" element={<Navigate to="." replace />} />
        </Routes>
    )
}
