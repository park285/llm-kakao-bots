import { useQuery } from '@tanstack/react-query'
import { logsApi, dockerApi } from '@/api'
import { queryKeys } from '@/api/queryKeys'
import { ScrollText, RefreshCw, AlertTriangle, Shield, Activity, ChevronDown, ChevronRight, Server, Terminal, LayoutList } from 'lucide-react'
import { useEffect, useState } from 'react'
import clsx from 'clsx'
import type { LogEntry } from '@/types'
import { LogTerminal } from '@/components/docker/LogTerminal'
import { useSSRData } from '@/hooks/useSSRData'

// --- 헬퍼 함수 & 서브 컴포넌트 ---

const formatDetailValue = (value: unknown): string => {
    if (value === null) return 'null'
    if (value === undefined) return 'undefined'

    switch (typeof value) {
        case 'string':
            return value
        case 'number':
        case 'boolean':
        case 'bigint':
            return String(value)
        case 'symbol':
            return value.toString()
        case 'function':
            return '[function]'
        case 'object':
            if (value instanceof Date) return value.toISOString()
            try { return JSON.stringify(value) } catch { return '[unserializable]' }
        default:
            return ''
    }
}

const LogItem = ({ log }: { log: LogEntry }) => {
    const [expanded, setExpanded] = useState(false)

    const getTypeConfig = (type: string) => {
        if (type.includes('error') || type.includes('fail')) return { icon: AlertTriangle, color: 'text-rose-600', bg: 'bg-rose-50', border: 'border-rose-100' }
        if (type.includes('auth')) return { icon: Shield, color: 'text-amber-600', bg: 'bg-amber-50', border: 'border-amber-100' }
        if (type.includes('system') || type.includes('watchdog')) return { icon: Server, color: 'text-slate-600', bg: 'bg-slate-100', border: 'border-slate-200' }
        return { icon: Activity, color: 'text-sky-600', bg: 'bg-sky-50', border: 'border-sky-100' }
    }

    const { icon: Icon, color, border } = getTypeConfig(log.type)
    const details = log.details ?? {}
    const hasDetails = Object.keys(details).length > 0

    return (
        <div className="group text-sm bg-white border-b border-slate-100 last:border-0 hover:bg-slate-50 transition-colors">
            <div
                className={clsx(
                    "grid grid-cols-[140px_1fr] md:grid-cols-[150px_140px_1fr] items-start gap-4 p-3 cursor-pointer",
                    expanded && "bg-slate-50"
                )}
                onClick={() => {
                    if (!hasDetails) return
                    setExpanded((prev) => !prev)
                }}
            >
                <div className="font-mono text-xs text-slate-400 pt-0.5">
                    {new Date(log.timestamp).toLocaleString('ko-KR', {
                        year: '2-digit', month: '2-digit', day: '2-digit',
                        hour: '2-digit', minute: '2-digit', second: '2-digit',
                        hour12: false
                    })}
                </div>
                <div className="hidden md:flex items-center gap-1.5">
                    <div className={clsx("p-1 rounded bg-white border shrink-0", border)}>
                        <Icon size={12} className={color} />
                    </div>
                    <span className={clsx("text-xs font-bold uppercase tracking-wide", color)}>
                        {log.type.replace(/_/g, ' ')}
                    </span>
                </div>
                <div className="min-w-0 flex items-center gap-2">
                    <div className="md:hidden shrink-0">
                        <Icon size={14} className={color} />
                    </div>
                    <span className="font-medium text-slate-700 truncate">{log.summary}</span>
                    {hasDetails && (
                        <div className="ml-auto text-slate-300">
                            {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                        </div>
                    )}
                </div>
            </div>
            {expanded && hasDetails && (
                <div className="px-3 pb-3 pl-[28px] md:pl-[306px]">
                    <div className="bg-slate-100 rounded-lg p-3 border border-slate-200 text-xs font-mono overflow-x-auto">
                        <table className="w-full text-left border-collapse">
                            <tbody>
                                {Object.entries(details).map(([key, value]) => (
                                    <tr key={key} className="border-b border-slate-200/50 last:border-0">
                                        <td className="py-1 pr-4 text-slate-500 font-semibold align-top whitespace-nowrap">{key}</td>
                                        <td className="py-1 text-slate-700 break-all">
                                            {formatDetailValue(value)}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}
        </div>
    )
}

// --- 메인 컴포넌트 ---

const LogsTab = () => {
    const [viewMode, setViewMode] = useState<'audit' | 'console'>('audit')
    const [selectedContainer, setSelectedContainer] = useState<string>('')

    // --- Audit 로그 데이터 fetching ---
    const { data: logsData, isLoading: logsLoading, refetch: refetchLogs, isRefetching: logsRefetching } = useQuery({
        queryKey: queryKeys.logs.all,
        queryFn: logsApi.get,
        refetchInterval: 5000,
        enabled: viewMode === 'audit'
    })

    // --- Data Fetching for Docker Containers (useSSRData 훅 활용) ---
    const ssrContainersData = useSSRData('containers', (data) =>
        data?.status === 'ok' && Array.isArray(data.containers)
            ? { status: data.status, containers: data.containers }
            : undefined
    )

    const { data: containersData } = useQuery({
        queryKey: queryKeys.docker.containers,
        queryFn: dockerApi.getContainers,
        refetchInterval: 15000,
        initialData: ssrContainersData,
        enabled: viewMode === 'console'
    })

    const containers = containersData?.containers ?? []
    const managedContainers = containers.filter(c => c.managed)

    // 선택된 컨테이너 없을 시 첫 번째 managed 컨테이너 자동 선택
    useEffect(() => {
        if (viewMode !== 'console') return
        if (selectedContainer) return
        const [first] = managedContainers
        if (!first) return
        setSelectedContainer(first.name)
    }, [managedContainers, selectedContainer, viewMode])

    return (
        <div className="space-y-4 max-w-full h-[calc(100vh-140px)] flex flex-col">
            {/* Top Toolbar / Tabs */}
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-white p-2 rounded-xl border border-slate-200 shadow-sm shrink-0">
                <div className="flex items-center gap-1 bg-slate-100 p-1 rounded-lg w-full sm:w-auto">
                    <button
                        onClick={() => { setViewMode('audit') }}
                        className={clsx(
                            "flex items-center gap-2 px-4 py-2 rounded-md text-sm font-bold transition-all flex-1 sm:flex-none justify-center",
                            viewMode === 'audit'
                                ? "bg-white text-indigo-600 shadow-sm"
                                : "text-slate-500 hover:text-slate-700 hover:bg-slate-200/50"
                        )}
                    >
                        <LayoutList size={16} />
                        활동 기록
                    </button>
                    <button
                        onClick={() => { setViewMode('console') }}
                        className={clsx(
                            "flex items-center gap-2 px-4 py-2 rounded-md text-sm font-bold transition-all flex-1 sm:flex-none justify-center",
                            viewMode === 'console'
                                ? "bg-white text-sky-600 shadow-sm"
                                : "text-slate-500 hover:text-slate-700 hover:bg-slate-200/50"
                        )}
                    >
                        <Terminal size={16} />
                        실시간 콘솔
                    </button>
                </div>

                {viewMode === 'audit' ? (
                    <div className="flex items-center gap-3 w-full sm:w-auto justify-end px-2">
                        <span className="text-xs text-slate-500 font-medium hidden sm:inline">
                            최근 {logsData?.logs.length ?? 0}개의 활동
                        </span>
                        <button
                            onClick={() => { void refetchLogs() }}
                            className="p-2 hover:bg-slate-50 rounded-lg text-slate-500 transition-colors border border-transparent hover:border-slate-200"
                            title="새로고침"
                        >
                            <RefreshCw size={18} className={logsRefetching ? "animate-spin text-indigo-500" : ""} />
                        </button>
                    </div>
                ) : (
                    <div className="flex items-center gap-3 w-full sm:w-auto px-2">
                        <span className="text-xs text-slate-500 font-medium whitespace-nowrap hidden sm:inline">대상 컨테이너:</span>
                        <select
                            value={selectedContainer}
                            onChange={(e) => { setSelectedContainer(e.target.value) }}
                            className="bg-slate-50 text-slate-700 text-sm font-medium rounded-lg px-3 py-2 border border-slate-200 focus:outline-none focus:ring-2 focus:ring-sky-500 w-full sm:w-auto"
                        >
                            {managedContainers.map(c => (
                                <option key={c.name} value={c.name}>
                                    {c.name} ({c.state})
                                </option>
                            ))}
                        </select>
                    </div>
                )}
            </div>

            {/* Main Content Area */}
            <div className="flex-1 min-h-0 bg-white rounded-xl shadow-sm border border-slate-200 flex flex-col overflow-hidden relative">

                {viewMode === 'audit' && (
                    <>
                        <div className="flex-1 overflow-auto bg-slate-50 scrollbar-thin scrollbar-thumb-slate-200 scrollbar-track-transparent">
                            {logsLoading ? (
                                <div className="flex flex-col items-center justify-center h-full text-slate-400 gap-2">
                                    <RefreshCw className="animate-spin opacity-50" />
                                    <span className="text-sm">로그를 불러오는 중...</span>
                                </div>
                            ) : (!logsData?.logs || logsData.logs.length === 0) ? (
                                <div className="flex flex-col items-center justify-center h-full text-slate-400 gap-3">
                                    <div className="w-12 h-12 bg-slate-100 rounded-full flex items-center justify-center">
                                        <ScrollText className="text-slate-300" />
                                    </div>
                                    <p className="text-sm font-medium">기록된 로그가 없습니다.</p>
                                </div>
                            ) : (
                                <div className="bg-white min-w-[600px] md:min-w-0">
                                    <div className="hidden md:grid md:grid-cols-[150px_140px_1fr] border-b border-slate-100 bg-slate-50/50 p-2 pl-3 text-xs font-bold text-slate-500 uppercase tracking-wider sticky top-0 z-10">
                                        <div>Timestamp</div>
                                        <div>Type</div>
                                        <div>Activity</div>
                                    </div>
                                    {[...(logsData.logs)].reverse().map((log, idx) => (
                                        <LogItem key={`${log.timestamp}-${String(idx)}`} log={log} />
                                    ))}
                                </div>
                            )}
                        </div>
                        <div className="p-2 border-t border-slate-100 bg-white text-[11px] text-slate-400 text-center shrink-0">
                            Updates automatically every 5s • Showing last 100 system events
                        </div>
                    </>
                )}

                {viewMode === 'console' && (
                    <div className="flex-1 p-0 bg-black flex flex-col min-h-0">
                        {/* LogTerminal 컴포넌트 (자체 포함형) */}
                        <LogTerminal containerName={selectedContainer} />
                    </div>
                )}
            </div>
        </div>
    )
}

export default LogsTab
