import { Routes, Route, Navigate } from 'react-router-dom'
import TurtleSoupDashboard from '@/components/gameBots/TurtleSoup/TurtleSoupDashboard'
import TurtleSoupSessionsTable from '@/components/gameBots/TurtleSoup/TurtleSoupSessionsTable'
// import TurtleSoupPuzzleList from '@/components/gameBots/TurtleSoup/TurtleSoupPuzzleList'
// import TurtleSoupArchives from '@/components/gameBots/TurtleSoup/TurtleSoupArchives'

export default function TurtleSoupPage() {
    return (
        <Routes>
            <Route index element={<TurtleSoupDashboard />} />
            <Route path="sessions" element={<TurtleSoupSessionsTable />} />
            {/* 
      <Route path="puzzles" element={<TurtleSoupPuzzleList />} />
      <Route path="archives" element={<TurtleSoupArchives />} /> 
      */}
            <Route path="*" element={<Navigate to="." replace />} />
        </Routes>
    )
}
