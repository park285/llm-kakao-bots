import { useQuery } from '@tanstack/react-query'
import { tracesApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'
import { X, RefreshCw, Clock, Layers, Activity } from 'lucide-react'
import { useMemo } from 'react'
import { buildSpanTree, formatDuration } from '@/components/traces/utils'
import type { Span, TraceDetailResponse } from '@/types'
import { SpanTimeline } from '@/components/traces/SpanTimeline'

interface TraceDetailModalProps {
    traceId: string
    onClose: () => void
}

/**
 * TraceDetailModal: 트레이스 상세 정보를 모달로 표시함
 */
export const TraceDetailModal = ({ traceId, onClose }: TraceDetailModalProps) => {
    const { data, isLoading } = useQuery<TraceDetailResponse>({
        queryKey: queryKeys.traces.detail(traceId),
        queryFn: () => tracesApi.getTrace(traceId),
    })

    // Span Tree 구축함
    const spanTree = useMemo(() => {
        if (!data?.spans) return []
        return buildSpanTree(data.spans)
    }, [data?.spans])

    // Trace 통계 계산함
    const stats = useMemo(() => {
        if (!data?.spans || data.spans.length === 0) {
            return { duration: 0, spanCount: 0, serviceCount: 0, errorCount: 0, traceStart: 0 }
        }

        const { spans } = data
        const minStart = Math.min(...spans.map((s: Span) => s.startTime))
        const maxEnd = Math.max(...spans.map((s: Span) => s.startTime + s.duration))
        const services = new Set(spans.map((s: Span) => s.serviceName))
        const errors = spans.filter((s: Span) => s.hasError).length

        return {
            duration: maxEnd - minStart,
            spanCount: spans.length,
            serviceCount: services.size,
            errorCount: errors,
            traceStart: minStart,
        }
    }, [data])

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-2xl shadow-2xl w-full max-w-6xl h-[90vh] flex flex-col">
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-slate-200 shrink-0">
                    <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-indigo-100 rounded-xl flex items-center justify-center">
                            <Activity className="w-5 h-5 text-indigo-600" />
                        </div>
                        <div>
                            <h2 className="text-lg font-bold text-slate-800">Trace Detail</h2>
                            <span className="text-xs font-mono text-slate-400">{traceId}</span>
                        </div>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-2 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                        <X size={20} className="text-slate-500" />
                    </button>
                </div>

                {/* Stats Bar */}
                <div className="flex items-center gap-6 px-6 py-3 bg-slate-50 border-b border-slate-200 shrink-0">
                    <div className="flex items-center gap-2">
                        <Clock size={16} className="text-slate-400" />
                        <span className="text-sm font-medium text-slate-600">
                            {formatDuration(stats.duration)}
                        </span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Layers size={16} className="text-slate-400" />
                        <span className="text-sm font-medium text-slate-600">
                            {stats.spanCount} Spans
                        </span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Activity size={16} className="text-slate-400" />
                        <span className="text-sm font-medium text-slate-600">
                            {stats.serviceCount} Services
                        </span>
                    </div>
                    {stats.errorCount > 0 && (
                        <div className="flex items-center gap-2 text-rose-600">
                            <span className="w-2 h-2 bg-rose-500 rounded-full" />
                            <span className="text-sm font-medium">
                                {stats.errorCount} Errors
                            </span>
                        </div>
                    )}
                </div>

                {/* Content */}
                <div className="flex-1 overflow-hidden p-4">
                    {isLoading ? (
                        <div className="flex flex-col items-center justify-center h-full text-slate-400 gap-2">
                            <RefreshCw className="animate-spin opacity-50" />
                            <span className="text-sm">로딩 중...</span>
                        </div>
                    ) : (
                        <SpanTimeline
                            spanTree={spanTree}
                            traceStart={stats.traceStart}
                            traceDuration={stats.duration * 1.05}
                        />
                    )}
                </div>
            </div>
        </div>
    )
}
