import { useTwentyQStats, useTwentyQCategoryStats } from '@/hooks/useGameBots'
import { StatCard } from '@/components/ui/StatCard'
import { Brain, Trophy, Users, PlayCircle, CheckCircle } from 'lucide-react'

export default function TwentyQDashboard() {
    const { data: statsData, isLoading: isStatsLoading } = useTwentyQStats()
    const { data: categoryData, isLoading: isCategoryLoading } = useTwentyQCategoryStats()

    if (isStatsLoading || isCategoryLoading) {
        return <div className="p-8 text-center text-slate-500">통계 데이터를 불러오는 중...</div>
    }

    const stats = statsData?.stats

    if (!stats) {
        return <div className="p-8 text-center text-slate-500">데이터가 없습니다.</div>
    }

    return (
        <div className="space-y-6">
            <h2 className="text-xl font-bold text-slate-800 flex items-center gap-2">
                <Brain className="w-6 h-6 text-sky-500" />
                스무고개 현황
            </h2>

            {/* 상단 통계 카드 */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard
                    label="활성 세션"
                    value={stats.activeSessions}
                    icon={<PlayCircle className="w-5 h-5 text-emerald-500" />}
                    variant="green"
                />
                <StatCard
                    label="총 참여자"
                    value={stats.totalParticipants}
                    icon={<Users className="w-5 h-5 text-blue-500" />}
                    variant="blue"
                />
                <StatCard
                    label="오늘 진행된 게임"
                    value={stats.last24HoursGames}
                    icon={<Trophy className="w-5 h-5 text-amber-500" />}
                    variant="yellow"
                />
                <StatCard
                    label="전체 성공률"
                    value={`${(stats.successRate * 100).toFixed(1)}%`}
                    icon={<CheckCircle className="w-5 h-5 text-indigo-500" />}
                    variant="indigo"
                />
            </div>

            {/* 상세 통계 */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* 게임 결과 분포 */}
                <div className="bg-white p-6 rounded-2xl shadow-sm border border-slate-100">
                    <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
                        <CheckCircle className="w-5 h-5 text-slate-500" />
                        게임 결과 분포
                    </h3>
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <span className="text-slate-600">정답 맞춤 (AI 승리)</span>
                            <span className="font-bold text-green-600">{stats.totalCorrect}회</span>
                        </div>
                        <div className="w-full bg-slate-100 rounded-full h-2">
                            <div
                                className="bg-green-500 h-2 rounded-full transition-all duration-500"
                                style={{ width: `${(stats.totalCorrect / stats.totalPlayed) * 100}%` }}
                            />
                        </div>

                        <div className="flex items-center justify-between">
                            <span className="text-slate-600">포기 (플레이어 승리)</span>
                            <span className="font-bold text-rose-600">{stats.totalSurrender}회</span>
                        </div>
                        <div className="w-full bg-slate-100 rounded-full h-2">
                            <div
                                className="bg-rose-500 h-2 rounded-full transition-all duration-500"
                                style={{ width: `${(stats.totalSurrender / stats.totalPlayed) * 100}%` }}
                            />
                        </div>
                    </div>
                </div>

                {/* 카테고리별 통계 */}
                <div className="bg-white p-6 rounded-2xl shadow-sm border border-slate-100">
                    <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
                        <Brain className="w-5 h-5 text-slate-500" />
                        인기 카테고리
                    </h3>
                    <div className="space-y-3 max-h-[300px] overflow-y-auto pr-2 scrollbar-thin">
                        {categoryData?.categories.map((cat) => (
                            <div key={cat.category} className="flex items-center justify-between p-3 bg-slate-50 rounded-xl hover:bg-slate-100 transition-colors">
                                <div>
                                    <div className="font-bold text-slate-700">{cat.category}</div>
                                    <div className="text-xs text-slate-500">{cat.totalGames}게임 진행</div>
                                </div>
                                <div className="text-right">
                                    <div className="text-sm font-bold text-sky-600">
                                        {cat.successCount > 0 ? ((cat.successCount / cat.totalGames) * 100).toFixed(0) : 0}% 정답
                                    </div>
                                    <div className="text-xs text-rose-500">
                                        {(cat.surrenderRate * 100).toFixed(0)}% 포기
                                    </div>
                                </div>
                            </div>
                        ))}
                        {(!categoryData?.categories || categoryData.categories.length === 0) && (
                            <div className="text-center text-slate-400 py-4">카테고리 데이터 없음</div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}
