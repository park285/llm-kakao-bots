import { useTurtleSoupStats } from '@/hooks/useGameBots'
import { StatCard } from '@/components/ui/StatCard'
import { Skeleton } from '@/components/ui/Skeleton'

import { Puzzle, CheckCircle, HelpCircle } from 'lucide-react'

export default function TurtleSoupDashboard() {
    const { data, isLoading } = useTurtleSoupStats()

    if (isLoading) {
        return (
            <div className="space-y-6">
                <div className="flex items-center gap-2">
                    <Skeleton className="w-6 h-6 rounded-full" />
                    <Skeleton className="h-7 w-32" />
                </div>

                {/* Stat Cards Skeleton */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                    {[...Array(4)].map((_, i) => (
                        <div key={i} className="bg-white rounded-2xl p-6 border border-slate-100 shadow-sm relative overflow-hidden">
                            <div className="flex justify-between items-start">
                                <div className="space-y-2">
                                    <Skeleton className="h-4 w-16" />
                                    <Skeleton className="h-8 w-24" />
                                </div>
                                <Skeleton className="h-12 w-12 rounded-xl" />
                            </div>
                        </div>
                    ))}
                </div>

                {/* Detailed Stats Skeleton */}
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {/* Solve Status Skeleton */}
                    <div className="bg-white p-6 rounded-2xl shadow-sm border border-slate-100">
                        <div className="flex items-center gap-2 mb-6">
                            <Skeleton className="w-5 h-5 rounded-full" />
                            <Skeleton className="h-6 w-32" />
                        </div>
                        <div className="space-y-6">
                            {[...Array(2)].map((_, i) => (
                                <div key={i} className="space-y-2">
                                    <div className="flex justify-between">
                                        <Skeleton className="h-4 w-24" />
                                        <Skeleton className="h-4 w-12" />
                                    </div>
                                    <Skeleton className="h-2 w-full rounded-full" />
                                </div>
                            ))}
                        </div>
                    </div>

                    {/* Hint Usage Skeleton */}
                    <div className="bg-white p-6 rounded-2xl shadow-sm border border-slate-100">
                        <div className="flex items-center gap-2 mb-6">
                            <Skeleton className="w-5 h-5 rounded-full" />
                            <Skeleton className="h-6 w-32" />
                        </div>
                        <div className="flex flex-col items-center justify-center py-8 space-y-2">
                            <Skeleton className="h-10 w-24" />
                            <Skeleton className="h-4 w-40" />
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    const stats = data?.stats

    if (!stats) {
        return <div className="p-8 text-center text-slate-500">데이터가 없습니다.</div>
    }

    return (
        <div className="space-y-6">
            <h2 className="text-xl font-bold text-slate-800 flex items-center gap-2">
                <Puzzle className="w-6 h-6 text-emerald-500" />
                바다거북스프 현황
            </h2>

            {/* 상단 통계 카드 */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard
                    label="활성 세션"
                    value={stats.activeSessions}
                    icon={<HelpCircle className="w-5 h-5 text-indigo-500" />}
                    variant="indigo"
                />
                <StatCard
                    label="성공률"
                    value={`${(stats.solveRate * 100).toFixed(1)}%`}
                    icon={<CheckCircle className="w-5 h-5 text-emerald-500" />}
                    variant="green"
                />
                <StatCard
                    label="평균 질문 수"
                    value={stats.avgQuestions.toFixed(1)}
                    icon={<HelpCircle className="w-5 h-5 text-amber-500" />}
                    variant="yellow"
                />
                <StatCard
                    label="오늘 해결됨"
                    value={stats.last24HoursSolve}
                    icon={<CheckCircle className="w-5 h-5 text-blue-500" />}
                    variant="blue"
                />
            </div>

            {/* 추가 통계 */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="bg-white p-6 rounded-2xl shadow-sm border border-slate-100">
                    <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
                        <CheckCircle className="w-5 h-5 text-slate-500" />
                        해결 현황
                    </h3>
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <span className="text-slate-600">해결 성공</span>
                            <span className="font-bold text-emerald-600">{stats.totalSolved}회</span>
                        </div>
                        <div className="w-full bg-slate-100 rounded-full h-2">
                            <div
                                className="bg-emerald-500 h-2 rounded-full"
                                style={{ width: `${(stats.totalSolved / (stats.totalSolved + stats.totalFailed || 1)) * 100}%` }}
                            />
                        </div>

                        <div className="flex items-center justify-between">
                            <span className="text-slate-600">실패/포기</span>
                            <span className="font-bold text-rose-600">{stats.totalFailed}회</span>
                        </div>
                        <div className="w-full bg-slate-100 rounded-full h-2">
                            <div
                                className="bg-rose-500 h-2 rounded-full"
                                style={{ width: `${(stats.totalFailed / (stats.totalSolved + stats.totalFailed || 1)) * 100}%` }}
                            />
                        </div>
                    </div>
                </div>

                <div className="bg-white p-6 rounded-2xl shadow-sm border border-slate-100">
                    <h3 className="text-lg font-bold text-slate-800 mb-4 flex items-center gap-2">
                        <HelpCircle className="w-5 h-5 text-slate-500" />
                        힌트 사용량
                    </h3>
                    <div className="flex flex-col items-center justify-center py-8">
                        <div className="text-4xl font-black text-amber-500 mb-2">
                            {stats.avgHintsPerGame.toFixed(2)}
                        </div>
                        <div className="text-sm text-slate-500 font-medium">게임 당 평균 힌트 사용</div>
                    </div>
                </div>
            </div>
        </div>
    )
}
