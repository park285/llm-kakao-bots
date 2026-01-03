import { useQuery } from '@tanstack/react-query'
import { tracesApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'
import {
    Search,
    RefreshCw,
    AlertTriangle,
    Activity,
    Clock,
    Filter,
} from 'lucide-react'
import { useState, useMemo } from 'react'
import clsx from 'clsx'
import type {
    TraceSummary,
    TraceSearchParams,
    ServicesResponse,
    OperationsResponse,
    TracesHealthResponse,
    TraceSearchResponse
} from '@/types'
import { TraceDetailModal } from '@/components/traces/TraceDetailModal'
import { formatDuration } from '@/components/traces/utils'

// 시간 범위 옵션
const lookbackOptions = [
    { value: '1h', label: '최근 1시간' },
    { value: '6h', label: '최근 6시간' },
    { value: '24h', label: '최근 24시간' },
    { value: '7d', label: '최근 7일' },
]

// TraceExplorer: 트레이스 검색 및 필터링, 목록 조회를 담당하는 컴포넌트입니다.
export const TraceExplorer = () => {
    // 필터 상태
    const [selectedService, setSelectedService] = useState<string>('')
    const [selectedOperation, setSelectedOperation] = useState<string>('')
    const [lookback, setLookback] = useState<string>('1h')
    const [limit, setLimit] = useState<number>(20)
    const [errorOnly, setErrorOnly] = useState<boolean>(false)

    // 상세 모달 상태
    const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null)

    // Jaeger 가용성 확인
    const { data: healthData } = useQuery<TracesHealthResponse>({
        queryKey: queryKeys.traces.health,
        queryFn: tracesApi.checkHealth,
        staleTime: 30000,
    })

    // setLimit 사용을 위한 더미 호출 (Lint 에러 방지용)
    void setLimit

    const isJaegerAvailable = healthData?.available ?? false

    // 서비스 목록 조회함
    const { data: servicesData, isLoading: servicesLoading } = useQuery<ServicesResponse>({
        queryKey: queryKeys.traces.services,
        queryFn: tracesApi.getServices,
        enabled: isJaegerAvailable,
    })

    // Operation 목록 조회함
    const { data: operationsData, isLoading: operationsLoading } = useQuery<OperationsResponse>({
        queryKey: queryKeys.traces.operations(selectedService),
        queryFn: () => tracesApi.getOperations(selectedService),
        enabled: isJaegerAvailable && !!selectedService,
    })

    // 검색 파라미터 메모이제이션
    const searchParams: TraceSearchParams = useMemo(() => ({
        service: selectedService,
        operation: selectedOperation || undefined,
        lookback,
        limit,
        tags: errorOnly ? { error: 'true' } : undefined,
    }), [selectedService, selectedOperation, lookback, limit, errorOnly])

    // 트레이스 검색함
    const {
        data: tracesData,
        isLoading: tracesLoading,
        refetch: refetchTraces,
        isRefetching,
    } = useQuery<TraceSearchResponse>({
        queryKey: queryKeys.traces.search(searchParams),
        queryFn: () => tracesApi.search(searchParams),
        enabled: isJaegerAvailable && !!selectedService,
        refetchInterval: 30000, // 30초마다 자동 갱신
    })

    const services = (servicesData?.services ?? []).filter(s =>
        !['jaeger-all-in-one', 'jaeger', 'admin-dashboard', 'admin-backend'].includes(s)
    )
    const operations = operationsData?.operations ?? []
    const traces = tracesData?.traces ?? []

    // Jaeger 비가용 시
    if (!isJaegerAvailable) {
        return (
            <div className="flex flex-col items-center justify-center h-64 text-slate-400 gap-3">
                <div className="w-16 h-16 bg-amber-100 rounded-full flex items-center justify-center">
                    <AlertTriangle className="w-8 h-8 text-amber-500" />
                </div>
                <p className="text-sm font-medium text-slate-600">Jaeger 서비스를 사용할 수 없습니다</p>
                <p className="text-xs text-slate-400">Jaeger 컨테이너가 실행 중인지 확인하세요</p>
            </div>
        )
    }

    return (
        <div className="space-y-4 max-w-full h-full flex flex-col">
            {/* Toolbar */}
            <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4 bg-white p-3 rounded-xl border border-slate-200 shadow-sm shrink-0">
                {/* Filters */}
                <div className="flex flex-wrap items-center gap-3 w-full lg:w-auto">
                    {/* Service Selector */}
                    <div className="flex items-center gap-2">
                        <Activity size={16} className="text-slate-400" />
                        <select
                            value={selectedService}
                            onChange={(e) => {
                                setSelectedService(e.target.value)
                                setSelectedOperation('') // Operation 초기화
                            }}
                            disabled={servicesLoading}
                            className="bg-slate-50 text-slate-700 text-sm font-medium rounded-lg px-3 py-2 border border-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500 min-w-[160px]"
                        >
                            <option value="">서비스 선택...</option>
                            {services.map(s => (
                                <option key={s} value={s}>{s}</option>
                            ))}
                        </select>
                    </div>

                    {/* Operation Selector */}
                    <div className="flex items-center gap-2">
                        <Filter size={16} className="text-slate-400" />
                        <select
                            value={selectedOperation}
                            onChange={(e) => { setSelectedOperation(e.target.value) }}
                            disabled={!selectedService || operationsLoading}
                            className="bg-slate-50 text-slate-700 text-sm font-medium rounded-lg px-3 py-2 border border-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500 min-w-[200px]"
                        >
                            <option value="">모든 Operation</option>
                            {operations.map(op => (
                                <option key={op} value={op}>{op}</option>
                            ))}
                        </select>
                    </div>

                    {/* Time Range */}
                    <div className="flex items-center gap-2">
                        <Clock size={16} className="text-slate-400" />
                        <select
                            value={lookback}
                            onChange={(e) => { setLookback(e.target.value) }}
                            className="bg-slate-50 text-slate-700 text-sm font-medium rounded-lg px-3 py-2 border border-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        >
                            {lookbackOptions.map(opt => (
                                <option key={opt.value} value={opt.value}>{opt.label}</option>
                            ))}
                        </select>
                    </div>

                    {/* Error Only Toggle */}
                    <label className="flex items-center gap-2 cursor-pointer">
                        <input
                            type="checkbox"
                            checked={errorOnly}
                            onChange={(e) => { setErrorOnly(e.target.checked) }}
                            className="w-4 h-4 rounded border-slate-300 text-rose-500 focus:ring-rose-500"
                        />
                        <span className="text-sm font-medium text-slate-600">에러만</span>
                    </label>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-3 w-full lg:w-auto justify-end">
                    <span className="text-xs text-slate-500 font-medium">
                        {traces.length}개 트레이스
                    </span>
                    <button
                        onClick={() => { void refetchTraces() }}
                        disabled={!selectedService}
                        className="p-2 hover:bg-slate-50 rounded-lg text-slate-500 transition-colors border border-transparent hover:border-slate-200 disabled:opacity-50"
                        title="새로고침"
                    >
                        <RefreshCw size={18} className={isRefetching ? 'animate-spin text-indigo-500' : ''} />
                    </button>
                </div>
            </div>

            {/* Trace List */}
            <div className="flex-1 min-h-0 bg-white rounded-xl shadow-sm border border-slate-200 flex flex-col overflow-hidden">
                {/* Table Header */}
                <div className="grid grid-cols-[1fr_2fr_100px_100px_80px] gap-4 px-4 py-3 border-b border-slate-100 bg-slate-50/50 text-xs font-bold text-slate-500 uppercase tracking-wider shrink-0">
                    <div>Trace ID</div>
                    <div>Operation</div>
                    <div>Duration</div>
                    <div>Time</div>
                    <div>Spans</div>
                </div>

                {/* Table Body */}
                <div className="flex-1 overflow-auto">
                    {tracesLoading ? (
                        <div className="flex flex-col items-center justify-center h-full text-slate-400 gap-2">
                            <RefreshCw className="animate-spin opacity-50" />
                            <span className="text-sm">트레이스를 불러오는 중...</span>
                        </div>
                    ) : !selectedService ? (
                        <div className="flex flex-col items-center justify-center h-full text-slate-400 gap-3">
                            <div className="w-12 h-12 bg-slate-100 rounded-full flex items-center justify-center">
                                <Search className="text-slate-300" />
                            </div>
                            <p className="text-sm font-medium">서비스를 선택하세요</p>
                        </div>
                    ) : traces.length === 0 ? (
                        <div className="flex flex-col items-center justify-center h-full text-slate-400 gap-3">
                            <div className="w-12 h-12 bg-slate-100 rounded-full flex items-center justify-center">
                                <Activity className="text-slate-300" />
                            </div>
                            <p className="text-sm font-medium">조건에 맞는 트레이스가 없습니다</p>
                        </div>
                    ) : (
                        traces.map((trace: TraceSummary) => (
                            <TraceRow
                                key={trace.traceId}
                                trace={trace}
                                onClick={() => { setSelectedTraceId(trace.traceId) }}
                            />
                        ))
                    )}
                </div>

                {/* Footer */}
                <div className="p-2 border-t border-slate-100 bg-white text-[11px] text-slate-400 text-center shrink-0">
                    자동 갱신 30초 • 시간 단위: Microseconds (μs)
                </div>
            </div>

            {/* Trace Detail Modal */}
            {selectedTraceId && (
                <TraceDetailModal
                    traceId={selectedTraceId}
                    onClose={() => { setSelectedTraceId(null) }}
                />
            )}
        </div>
    )
}

// Trace Row 컴포넌트
interface TraceRowProps {
    trace: TraceSummary
    onClick: () => void
}

const TraceRow = ({ trace, onClick }: TraceRowProps) => {
    const formattedTime = new Date(trace.startTime).toLocaleTimeString('ko-KR', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
    })

    return (
        <div
            onClick={onClick}
            className={clsx(
                'grid grid-cols-[1fr_2fr_100px_100px_80px] gap-4 px-4 py-3',
                'border-b border-slate-100 last:border-0',
                'hover:bg-slate-50 cursor-pointer transition-colors',
                trace.hasError && 'bg-rose-50/50 hover:bg-rose-50'
            )}
        >
            {/* Trace ID */}
            <div className="flex items-center gap-2 min-w-0">
                {trace.hasError && (
                    <AlertTriangle size={14} className="text-rose-500 shrink-0" />
                )}
                <span className="font-mono text-xs text-slate-600 truncate">
                    {trace.traceId.substring(0, 12)}...
                </span>
            </div>

            {/* Operation */}
            <div className="flex items-center gap-2 min-w-0">
                <span className="font-medium text-sm text-slate-700 truncate">
                    {trace.operationName}
                </span>
                <div className="flex gap-1 shrink-0">
                    {trace.services.slice(0, 3).map(svc => (
                        <span
                            key={svc}
                            className="px-1.5 py-0.5 text-[10px] font-medium bg-slate-100 text-slate-500 rounded"
                        >
                            {svc}
                        </span>
                    ))}
                    {trace.services.length > 3 && (
                        <span className="text-[10px] text-slate-400">
                            +{trace.services.length - 3}
                        </span>
                    )}
                </div>
            </div>

            {/* Duration */}
            <div className={clsx(
                'text-sm font-mono',
                trace.duration > 1000000 ? 'text-amber-600 font-semibold' : 'text-slate-600'
            )}>
                {formatDuration(trace.duration)}
            </div>

            {/* Time */}
            <div className="text-xs text-slate-400 font-mono">
                {formattedTime}
            </div>

            {/* Span Count */}
            <div className="text-sm text-slate-500">
                {trace.spanCount}
            </div>
        </div>
    )
}
