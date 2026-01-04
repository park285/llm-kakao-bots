import { useState } from 'react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts'
import { useWebSocket } from '@/hooks/useWebSocket'
import type { SystemStats } from '@/types'
import { CONFIG } from '@/config'
import { Card, Badge } from '@/components/ui'
import { Activity, Cpu, CircuitBoard, Layers, Server } from 'lucide-react'
import * as z from 'zod'
import { cn } from '@/lib/utils'

// chart 데이터를 위한 확장 인터페이스
interface SystemStatsPoint extends SystemStats {
    time: string
    timestamp: number
}

const MAX_DATA_POINTS = 30

const systemStatsSchema = z.object({
    cpuUsage: z.coerce.number(),
    memoryUsage: z.coerce.number(),
    memoryTotal: z.coerce.number(),
    memoryUsed: z.coerce.number(),
    goroutines: z.coerce.number(),
    totalGoroutines: z.coerce.number(),
    serviceGoroutines: z.array(z.object({
        name: z.string(),
        goroutines: z.coerce.number(),
        available: z.boolean(),
    })),
})

type TooltipPayloadItem = {
    value?: number | string
    dataKey?: string | number
    payload: SystemStatsPoint
    color?: string
}

type CustomTooltipProps = {
    active?: boolean
    payload?: TooltipPayloadItem[]
    label?: string
}

// 컴포넌트 외부 정의로 리렌더링 시 함수 재생성 방지
const CustomTooltip = ({ active, payload, label }: CustomTooltipProps) => {
    if (active && payload && payload.length > 0) {
        const cpuValue = payload[0]?.value
        const memoryValue = payload[1]?.value
        const cpu = typeof cpuValue === 'number' ? cpuValue : Number(cpuValue)
        const memory = typeof memoryValue === 'number' ? memoryValue : Number(memoryValue)

        // payload에서 goroutine 데이터 추출함 (chart에 없을 수 있음)
        // 원본 데이터는 point를 통해 접근 가능
        const stats = payload[0]?.payload as SystemStatsPoint

        return (
            <div className="bg-white p-3 border border-slate-200 shadow-lg rounded-lg text-xs min-w-[160px]">
                <p className="font-bold text-slate-700 mb-2 border-b border-slate-100 pb-1">{label}</p>
                <div className="space-y-1.5">
                    {/* CPU & Memory (헤더 차트) */}
                    {payload.some(p => p.dataKey === 'cpuUsage') && (
                        <>
                            <div className="flex justify-between items-center gap-4">
                                <span className="text-sky-600 font-medium flex items-center gap-1">
                                    <Cpu size={10} /> CPU
                                </span>
                                <span className="font-mono">{Number.isFinite(cpu) ? `${cpu.toFixed(1)}%` : '-'}</span>
                            </div>
                            <div className="flex justify-between items-center gap-4">
                                <span className="text-violet-600 font-medium flex items-center gap-1">
                                    <Layers size={10} /> Memory
                                </span>
                                <span className="font-mono">{Number.isFinite(memory) ? `${memory.toFixed(1)}%` : '-'}</span>
                            </div>
                        </>
                    )}

                    {/* Goroutine 분석 (Stacked Area Chart) */}
                    {payload.some(p => CONFIG.ui.serviceColors[p.dataKey as string]) && (
                        <div className="space-y-1 mt-1">
                            <p className="text-[10px] text-slate-400 font-bold uppercase tracking-tighter mb-1">서비스별 고루틴</p>
                            {payload.filter(p => CONFIG.ui.serviceColors[p.dataKey as string]).map((p) => (
                                <div key={p.dataKey} className="flex justify-between items-center gap-4">
                                    <span className="flex items-center gap-1.5" style={{ color: p.color }}>
                                        <div className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: p.color }} />
                                        {p.dataKey}
                                    </span>
                                    <span className="font-mono font-bold">{p.value}</span>
                                </div>
                            ))}
                            <div className="flex justify-between items-center gap-4 border-t border-slate-50 pt-1 mt-1 font-bold">
                                <span className="text-slate-500">Total</span>
                                <span className="font-mono">{stats.totalGoroutines}</span>
                            </div>
                        </div>
                    )}

                    {/* 분석 차트가 아닌 경우 일반 통계 표시 */}
                    {!payload.some(p => CONFIG.ui.serviceColors[p.dataKey as string]) && stats && (
                        <div className="flex justify-between items-center gap-4 border-t border-slate-50 pt-1.5 mt-1.5">
                            <span className="text-slate-500 font-medium flex items-center gap-1">
                                <CircuitBoard size={10} /> Goroutines
                            </span>
                            <span className="font-mono text-slate-700">{stats.totalGoroutines}</span>
                        </div>
                    )}
                </div>
            </div>
        )
    }
    return null
}

export const SystemStatsChart = () => {
    const [statsHistory, setStatsHistory] = useState<SystemStatsPoint[]>([])
    const [currentStats, setCurrentStats] = useState<SystemStats | null>(null)

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/admin/api/ws/system-stats`

    const { isConnected } = useWebSocket<SystemStats>(wsUrl, {
        parseMessage: (data) => {
            const parsed = systemStatsSchema.safeParse(data)
            return parsed.success ? parsed.data : null
        },
        onMessage: (data) => {
            const now = new Date()
            const timeStr = now.toLocaleTimeString('ko-KR', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })

            // serviceGoroutines를 chart 접근용 map으로 변환함
            const serviceMap: Record<string, number> = {}
            data.serviceGoroutines.forEach(svc => {
                serviceMap[svc.name] = svc.available ? svc.goroutines : 0
            })

            const point: SystemStatsPoint = {
                ...data,
                ...serviceMap, // Flatten for chart
                time: timeStr,
                timestamp: now.getTime()
            }

            setCurrentStats(data)
            setStatsHistory(prev => {
                const newHistory = [...prev, point]
                return newHistory.slice(-MAX_DATA_POINTS)
            })
        },
        reconnectInterval: 5000
    })

    return (
        <Card className="overflow-hidden">
            <Card.Header className="flex flex-row items-center justify-between border-b border-slate-100 pb-4 bg-slate-50/50">
                <div className="flex items-center gap-2">
                    <Activity className="text-slate-500" size={20} />
                    <h3 className="text-lg font-bold text-slate-800">시스템 리소스</h3>
                    {isConnected ? (
                        <span className="flex h-2 w-2 relative ml-2">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                        </span>
                    ) : (
                        <span className="h-2 w-2 rounded-full bg-slate-300 ml-2"></span>
                    )}
                </div>

                {currentStats && (
                    <div className="flex gap-4 text-xs font-mono">
                        <div className="flex items-center gap-1.5 px-2 py-1 bg-white rounded border border-slate-100 shadow-sm">
                            <Cpu size={14} className="text-sky-500" />
                            <span className="font-bold text-slate-700">{currentStats.cpuUsage.toFixed(1)}%</span>
                        </div>
                        <div className="flex items-center gap-1.5 px-2 py-1 bg-white rounded border border-slate-100 shadow-sm">
                            <Layers size={14} className="text-violet-500" />
                            <span className="font-bold text-slate-700">{currentStats.memoryUsage.toFixed(1)}%</span>
                        </div>
                        <div className="flex items-center gap-1.5 px-2 py-1 bg-white rounded border border-slate-100 shadow-sm hidden sm:flex">
                            <CircuitBoard size={14} className="text-slate-400" />
                            <span className="font-bold text-slate-500">{currentStats.totalGoroutines} Goroutines</span>
                        </div>
                    </div>
                )}
            </Card.Header>

            <Card.Body className="p-0 relative">
                {/* 데이터 수집 중 로딩 오버레이 (선이 그려질 때까지 대기) */}
                {statsHistory.length < 2 && (
                    <div className="absolute inset-0 flex items-center justify-center bg-slate-50/50 z-10 rounded-b-lg">
                        <div className="flex items-center gap-2">
                            <div className="h-4 w-4 border-2 border-slate-300 border-t-sky-500 rounded-full animate-spin" />
                            <span className="text-xs text-slate-500">데이터 수집 중...</span>
                        </div>
                    </div>
                )}
                <div className="w-full h-[200px] mt-4">
                    <ResponsiveContainer width="100%" height="100%">
                        <AreaChart data={statsHistory} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                            <defs>
                                <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="5%" stopColor="#0ea5e9" stopOpacity={0.3} />
                                    <stop offset="95%" stopColor="#0ea5e9" stopOpacity={0} />
                                </linearGradient>
                                <linearGradient id="colorMem" x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3} />
                                    <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} />
                                </linearGradient>
                            </defs>
                            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                            <XAxis
                                dataKey="time"
                                tick={{ fontSize: 10, fill: '#94a3b8' }}
                                tickLine={false}
                                axisLine={false}
                                interval="preserveStartEnd"
                                minTickGap={30}
                            />
                            <YAxis
                                domain={[0, 'auto']}
                                tick={{ fontSize: 10, fill: '#94a3b8' }}
                                tickLine={false}
                                axisLine={false}
                                tickFormatter={(value: number | string) => `${String(value)}%`}
                                width={40}
                            />
                            <Tooltip content={<CustomTooltip />} />
                            <Area
                                type="monotone"
                                dataKey="cpuUsage"
                                stroke="#0ea5e9"
                                strokeWidth={2}
                                fillOpacity={1}
                                fill="url(#colorCpu)"
                                isAnimationActive={false}
                            />
                            <Area
                                type="monotone"
                                dataKey="memoryUsage"
                                stroke="#8b5cf6"
                                strokeWidth={2}
                                fillOpacity={1}
                                fill="url(#colorMem)"
                                isAnimationActive={false}
                            />
                        </AreaChart>
                    </ResponsiveContainer>
                </div>

                <div className="px-4 py-3 border-t border-slate-100">
                    <div className="flex items-center gap-2 mb-4">
                        <CircuitBoard size={16} className="text-slate-400" />
                        <h4 className="text-sm font-bold text-slate-700">서비스별 고루틴</h4>
                    </div>
                    <div className="w-full h-[160px]">
                        <ResponsiveContainer width="100%" height="100%">
                            <AreaChart data={statsHistory} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                                <XAxis
                                    dataKey="time"
                                    tick={{ fontSize: 10, fill: '#94a3b8' }}
                                    tickLine={false}
                                    axisLine={false}
                                    interval="preserveStartEnd"
                                    minTickGap={30}
                                />
                                <YAxis
                                    domain={[0, 'auto']}
                                    tick={{ fontSize: 10, fill: '#94a3b8' }}
                                    tickLine={false}
                                    axisLine={false}
                                    width={30}
                                />
                                <Tooltip content={<CustomTooltip />} />
                                {currentStats?.serviceGoroutines.map((svc) => (
                                    <Area
                                        key={svc.name}
                                        type="monotone"
                                        dataKey={svc.name}
                                        stackId="goroutines"
                                        stroke={CONFIG.ui.serviceColors[svc.name] || '#64748b'}
                                        fill={CONFIG.ui.serviceColors[svc.name] || '#64748b'}
                                        fillOpacity={0.6}
                                        isAnimationActive={false}
                                    />
                                ))}
                            </AreaChart>
                        </ResponsiveContainer>
                    </div>
                </div>

                {currentStats && currentStats.serviceGoroutines && (
                    <div className="px-4 py-3 bg-slate-50/50 border-t border-slate-100">
                        <div className="flex items-center gap-2 mb-2">
                            <Server size={14} className="text-slate-400" />
                            <span className="text-xs font-bold text-slate-600 uppercase tracking-wider">Service Status</span>
                        </div>
                        <div className="flex gap-2 flex-wrap">
                            {currentStats.serviceGoroutines.map((svc) => (
                                <Badge
                                    key={svc.name}
                                    variant="outline"
                                    className="text-[10px] py-0 px-2 h-5 font-mono bg-white"
                                >
                                    <span
                                        className={cn("mr-1.5 h-1.5 w-1.5 rounded-full", svc.available ? "animate-pulse" : "bg-red-500")}
                                        style={{ backgroundColor: svc.available ? CONFIG.ui.serviceColors[svc.name] : undefined }}
                                    />
                                    <span style={{ color: svc.available ? CONFIG.ui.serviceColors[svc.name] : undefined, fontWeight: 600 }}>
                                        {svc.name}
                                    </span>
                                    <span className="text-slate-600 ml-1">
                                        : {svc.available ? svc.goroutines : 'OFFLINE'}
                                    </span>
                                </Badge>
                            ))}
                        </div>
                    </div>
                )}
            </Card.Body>
        </Card>
    )
}
