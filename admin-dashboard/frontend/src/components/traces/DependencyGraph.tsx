import { useQuery } from '@tanstack/react-query'
import { tracesApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'
import { useState } from 'react'
import { ArrowRight, Box, RefreshCw, Network } from 'lucide-react'
import type { DependenciesResponse } from '@/types'

// DependencyGraph: 서비스 간 호출 관계(의존성)를 목록 형태로 시각화하는 컴포넌트입니다.
// D3 등 그래프 라이브러리 의존성 없이 명확한 관계 파악을 위해 리스트 뷰를 제공합니다.
export const DependencyGraph = () => {
    const [lookback, setLookback] = useState<string>('24h')

    // 의존성 데이터 조회함
    const { data: dependencyData, isLoading, refetch, isRefetching } = useQuery<DependenciesResponse>({
        queryKey: queryKeys.traces.dependencies(lookback),
        queryFn: () => tracesApi.getDependencies(lookback),
        refetchInterval: 60000,
    })

    const dependencies = dependencyData?.dependencies ?? []

    // 호출 횟수 기준 내림차순 정렬함
    const sortedDependencies = [...dependencies].sort((a, b) => b.callCount - a.callCount)

    return (
        <div className="space-y-4 h-full flex flex-col">
            {/* Toolbar */}
            <div className="flex items-center justify-between bg-white p-3 rounded-xl border border-slate-200 shadow-sm shrink-0">
                <div className="flex items-center gap-2">
                    <Network size={16} className="text-slate-400" />
                    <select
                        value={lookback}
                        onChange={(e) => { setLookback(e.target.value) }}
                        className="bg-slate-50 text-slate-700 text-sm font-medium rounded-lg px-3 py-2 border border-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    >
                        <option value="1h">최근 1시간</option>
                        <option value="6h">최근 6시간</option>
                        <option value="24h">최근 24시간</option>
                        <option value="7d">최근 7일</option>
                    </select>
                </div>

                <div className="flex items-center gap-3">
                    <span className="text-xs text-slate-500 font-medium">
                        {dependencies.length}개 관계
                    </span>
                    <button
                        onClick={() => { void refetch() }}
                        className="p-2 hover:bg-slate-50 rounded-lg text-slate-500 transition-colors border border-transparent hover:border-slate-200"
                        title="새로고침"
                    >
                        <RefreshCw size={18} className={isRefetching ? 'animate-spin text-indigo-500' : ''} />
                    </button>
                </div>
            </div>

            {/* Content */}
            {isLoading ? (
                <div className="flex-1 flex flex-col items-center justify-center text-slate-400 gap-2">
                    <RefreshCw className="animate-spin opacity-50" />
                    <span className="text-sm">의존성 분석 중...</span>
                </div>
            ) : dependencies.length === 0 ? (
                <div className="flex-1 flex flex-col items-center justify-center text-slate-400 gap-3 border border-dashed border-slate-200 rounded-xl bg-slate-50/50">
                    <Network size={48} className="text-slate-200" />
                    <p className="text-sm font-medium">발견된 서비스 의존성이 없습니다</p>
                </div>
            ) : (
                <div className="flex-1 overflow-auto space-y-3 p-1">
                    {sortedDependencies.map((dep, idx) => (
                        <div
                            key={`${dep.parent}-${dep.child}-${String(idx)}`}
                            className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm flex items-center justify-between hover:border-indigo-200 transition-colors"
                        >
                            <div className="flex items-center gap-6">
                                {/* Parent Service */}
                                <div className="flex items-center gap-3 w-48 justify-end">
                                    <span className="font-medium text-slate-700 truncate" title={dep.parent}>
                                        {dep.parent}
                                    </span>
                                    <div className="w-8 h-8 rounded-lg bg-indigo-50 flex items-center justify-center shrink-0">
                                        <Box size={16} className="text-indigo-500" />
                                    </div>
                                </div>

                                {/* Flow Arrow */}
                                <div className="flex flex-col items-center gap-1 w-32">
                                    <span className="text-xs font-mono font-bold text-slate-400 bg-slate-100 px-2 py-0.5 rounded-full">
                                        {dep.callCount.toLocaleString()} calls
                                    </span>
                                    <ArrowRight size={20} className="text-slate-300" />
                                </div>

                                {/* Child Service */}
                                <div className="flex items-center gap-3 w-48">
                                    <div className="w-8 h-8 rounded-lg bg-emerald-50 flex items-center justify-center shrink-0">
                                        <Box size={16} className="text-emerald-500" />
                                    </div>
                                    <span className="font-medium text-slate-700 truncate" title={dep.child}>
                                        {dep.child}
                                    </span>
                                </div>
                            </div>

                            {/* Detail Button (Optional, maybe link to traces search) */}
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}
