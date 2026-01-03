/**
 * 채널 통계 테이블 컴포넌트
 */

import { useQuery } from '@tanstack/react-query'
import { statsApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'

export const ChannelStatsTable = () => {
    const { data: response, isLoading, isError, error } = useQuery({
        queryKey: queryKeys.stats.channels,
        queryFn: statsApi.getChannels,
        refetchInterval: 60000,
    })

    if (isLoading) {
        return <div className="text-center text-slate-400 py-8">채널 통계 로딩 중...</div>
    }

    if (isError) {
        return (
            <div className="text-center text-rose-500 py-8 bg-rose-50 rounded-lg border border-rose-100">
                <p className="font-medium">채널 통계를 불러올 수 없습니다</p>
                <p className="text-xs text-rose-400 mt-1">
                    {error instanceof Error ? error.message : 'Unknown error'}
                </p>
            </div>
        )
    }

    const stats = response?.stats ?? {}
    const sortedStats = Object.values(stats)
        .sort((a, b) => b.SubscriberCount - a.SubscriberCount)
        .slice(0, 10)

    if (sortedStats.length === 0) {
        return <div className="text-center text-slate-400 py-8">표시할 채널 통계가 없습니다</div>
    }

    return (
        <div className="overflow-x-auto rounded-lg border border-slate-100">
            <table className="w-full text-sm text-left">
                <thead className="text-xs text-slate-500 uppercase bg-slate-50 border-b border-slate-100">
                    <tr>
                        <th className="px-4 py-3 font-medium w-10">#</th>
                        <th className="px-4 py-3 font-medium">채널명</th>
                        <th className="px-4 py-3 font-medium text-right">구독자 수</th>
                        <th className="px-4 py-3 font-medium text-right">총 조회수</th>
                        <th className="px-4 py-3 font-medium text-right">동영상 수</th>
                    </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                    {sortedStats.map((stat, idx) => (
                        <tr key={stat.ChannelID} className="bg-white hover:bg-slate-50 transition-colors">
                            <td className="px-4 py-4 text-slate-400 font-bold">{idx + 1}</td>
                            <td className="px-4 py-4 font-medium text-slate-900">{stat.ChannelTitle}</td>
                            <td className="px-4 py-4 text-right text-slate-700 font-medium">
                                {stat.SubscriberCount.toLocaleString()}
                            </td>
                            <td className="px-4 py-4 text-right text-slate-500">
                                {stat.ViewCount.toLocaleString()}
                            </td>
                            <td className="px-4 py-4 text-right text-slate-500">
                                {stat.VideoCount.toLocaleString()}
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    )
}
