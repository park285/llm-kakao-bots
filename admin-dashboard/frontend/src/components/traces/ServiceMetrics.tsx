import { useQuery } from '@tanstack/react-query'
import { tracesApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'
import {
    LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer
} from 'recharts'
import { useState } from 'react'
import { RefreshCw, Activity } from 'lucide-react'
import clsx from 'clsx'
import type { ServicesResponse, ServiceMetricsResponse } from '@/types'

// ServiceMetrics: 서비스별 RED(Rate, Errors, Duration) 메트릭을 시각화하는 컴포넌트입니다.
// Jaeger SPM(Service Performance Monitoring) 데이터를 기반으로 차트를 렌더링합니다.
export const ServiceMetrics = () => {
    const [selectedService, setSelectedService] = useState<string>('')
    const [lookback, setLookback] = useState<string>('1h')

    // 서비스 목록 조회를 위해 API 호출함
    const { data: servicesData } = useQuery<ServicesResponse>({
        queryKey: queryKeys.traces.services,
        queryFn: tracesApi.getServices,
    })

    const services = (servicesData?.services ?? []).filter(s =>
        !['jaeger-all-in-one', 'jaeger', 'admin-dashboard', 'admin-backend'].includes(s)
    )

    // 선택된 서비스의 메트릭 데이터를 조회함 (30초 자동 갱신)
    const { data: metricsData, isLoading, refetch, isRefetching } = useQuery<ServiceMetricsResponse>({
        queryKey: queryKeys.traces.metrics(selectedService, { lookback }),
        queryFn: () => tracesApi.getMetrics(selectedService, { lookback }),
        enabled: !!selectedService,
        refetchInterval: 30000,
    })

    const metrics = metricsData?.metrics
    const latencies = metricsData?.latencies ?? []
    const calls = metricsData?.calls ?? []

    // 차트 데이터 포맷팅함 (Unix Timestamp -> 시간 문자열)
    // 조회 범위에 따라 포맷을 다르게 적용 (24h, 7d일 경우 날짜 포함, 나머지는 시간만 24시간제로 표시)
    const formatTime = (ts: number) => {
        const date = new Date(ts)
        const timeStr = `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`

        if (lookback === '24h' || lookback === '7d') {
            return `${date.getMonth() + 1}/${date.getDate()} ${timeStr}`
        }
        return timeStr
    }

    return (
        <div className="space-y-4 h-full flex flex-col">
            {/* Toolbar */}
            <div className="flex items-center justify-between bg-white p-3 rounded-xl border border-slate-200 shadow-sm shrink-0">
                <div className="flex items-center gap-3">
                    <div className="flex items-center gap-2">
                        <Activity size={16} className="text-slate-400" />
                        <select
                            value={selectedService}
                            onChange={(e) => { setSelectedService(e.target.value) }}
                            className="bg-slate-50 text-slate-700 text-sm font-medium rounded-lg px-3 py-2 border border-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500 min-w-[200px]"
                        >
                            <option value="">서비스 선택...</option>
                            {services.map(s => (
                                <option key={s} value={s}>{s}</option>
                            ))}
                        </select>
                    </div>

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

                <button
                    onClick={() => { void refetch() }}
                    disabled={!selectedService}
                    className="p-2 hover:bg-slate-50 rounded-lg text-slate-500 transition-colors border border-transparent hover:border-slate-200 disabled:opacity-50"
                    title="새로고침"
                >
                    <RefreshCw size={18} className={isRefetching ? 'animate-spin text-indigo-500' : ''} />
                </button>
            </div>

            {/* Content */}
            {!selectedService ? (
                <div className="flex-1 flex flex-col items-center justify-center text-slate-400 gap-3 border border-dashed border-slate-200 rounded-xl bg-slate-50/50">
                    <Activity size={48} className="text-slate-200" />
                    <p className="text-sm font-medium">분석할 서비스를 선택하세요</p>
                </div>
            ) : isLoading ? (
                <div className="flex-1 flex flex-col items-center justify-center text-slate-400 gap-2">
                    <RefreshCw className="animate-spin opacity-50" />
                    <span className="text-sm">메트릭 분석 중...</span>
                </div>
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 flex-1 overflow-auto p-1">
                    {/* Summary Cards */}
                    <MetricCard
                        label="Call Rate"
                        value={`${(metrics?.callRate ?? 0).toFixed(2)} ops/s`}
                        color="text-emerald-600"
                    />
                    <MetricCard
                        label="Error Rate"
                        value={`${((metrics?.errorRate ?? 0) * 100).toFixed(2)}%`}
                        color={((metrics?.errorRate ?? 0) > 0.01) ? "text-rose-600" : "text-slate-700"}
                    />
                    <MetricCard
                        label="P95 Latency"
                        value={`${(metrics?.p95Latency ?? 0).toFixed(2)} ms`}
                        color="text-indigo-600"
                    />

                    {/* Charts */}
                    <div className="lg:col-span-3 bg-white p-4 rounded-xl border border-slate-200 shadow-sm h-[300px]">
                        <h3 className="text-sm font-bold text-slate-700 mb-4 flex items-center gap-2">
                            <span className="w-2 h-2 rounded-full bg-indigo-500" />
                            Latency Trend (ms)
                        </h3>
                        <ResponsiveContainer width="100%" height="100%">
                            <LineChart data={latencies}>
                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#E2E8F0" />
                                <XAxis
                                    dataKey="timestamp"
                                    tickFormatter={(val) => formatTime(Number(val))}
                                    tick={{ fontSize: 11, fill: '#64748B' }}
                                    axisLine={false}
                                    tickLine={false}
                                    minTickGap={50}
                                />
                                <YAxis
                                    tick={{ fontSize: 11, fill: '#64748B' }}
                                    axisLine={false}
                                    tickLine={false}
                                />
                                <Tooltip
                                    labelFormatter={(label) => new Date(Number(label)).toLocaleString('ko-KR')}
                                    contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                                    formatter={(value: number | undefined) => [
                                        typeof value === 'number' ? `${value.toFixed(2)} ms` : '-',
                                        'Latency',
                                    ]}
                                />
                                <Line
                                    type="monotone"
                                    dataKey="value"
                                    stroke="#6366F1"
                                    strokeWidth={2}
                                    dot={false}
                                    activeDot={{ r: 4 }}
                                />
                            </LineChart>
                        </ResponsiveContainer>
                    </div>

                    <div className="lg:col-span-3 bg-white p-4 rounded-xl border border-slate-200 shadow-sm h-[300px]">
                        <h3 className="text-sm font-bold text-slate-700 mb-4 flex items-center gap-2">
                            <span className="w-2 h-2 rounded-full bg-emerald-500" />
                            Request Rate (ops/s)
                        </h3>
                        <ResponsiveContainer width="100%" height="100%">
                            <BarChart data={calls}>
                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#E2E8F0" />
                                <XAxis
                                    dataKey="timestamp"
                                    tickFormatter={(val) => formatTime(Number(val))}
                                    tick={{ fontSize: 11, fill: '#64748B' }}
                                    axisLine={false}
                                    tickLine={false}
                                    minTickGap={50}
                                />
                                <YAxis
                                    tick={{ fontSize: 11, fill: '#64748B' }}
                                    axisLine={false}
                                    tickLine={false}
                                />
                                <Tooltip
                                    labelFormatter={(label) => new Date(Number(label)).toLocaleString('ko-KR')}
                                    cursor={{ fill: '#F1F5F9' }}
                                    contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                                    formatter={(value: number | undefined) => [
                                        typeof value === 'number' ? `${value.toFixed(2)} ops/s` : '-',
                                        'Rate',
                                    ]}
                                />
                                <Bar dataKey="value" fill="#10B981" radius={[4, 4, 0, 0]} />
                            </BarChart>
                        </ResponsiveContainer>
                    </div>
                </div>
            )}
        </div>
    )
}

// MetricCard: 단일 메트릭 수치를 표시하는 카드 컴포넌트입니다.
const MetricCard = ({ label, value, color }: { label: string, value: string, color: string }) => (
    <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm">
        <div className="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">{label}</div>
        <div className={clsx("text-2xl font-bold font-mono", color)}>{value}</div>
    </div>
)
