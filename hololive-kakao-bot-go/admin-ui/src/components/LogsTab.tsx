import { useQuery } from '@tanstack/react-query'
import { logsApi } from '../api'
import { ScrollText, RefreshCw, AlertTriangle, Shield, Activity, ChevronDown, ChevronRight, Server } from 'lucide-react'
import { useState } from 'react'
import clsx from 'clsx'
import type { LogEntry } from '../types'

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
            if (value instanceof Date) {
                return value.toISOString()
            }
            try {
                return JSON.stringify(value)
            } catch {
                return '[unserializable]'
            }
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
                {/* 1. Time */}
                <div className="font-mono text-xs text-slate-400 pt-0.5">
                    {new Date(log.timestamp).toLocaleString('ko-KR', {
                        year: '2-digit', month: '2-digit', day: '2-digit',
                        hour: '2-digit', minute: '2-digit', second: '2-digit',
                        hour12: false
                    })}
                </div>

                {/* 2. Type (Mobile: hidden or merged, Desktop: column) */}
                <div className="hidden md:flex items-center gap-1.5">
                    <div className={clsx("p-1 rounded bg-white border shrink-0", border)}>
                        <Icon size={12} className={color} />
                    </div>
                    <span className={clsx("text-xs font-bold uppercase tracking-wide", color)}>
                        {log.type.replace(/_/g, ' ')}
                    </span>
                </div>

                {/* 3. Summary & Details Indicator */}
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

            {/* Expanded Details */}
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

const LogsTab = () => {
    const { data: logsData, isLoading, refetch, isRefetching } = useQuery({
        queryKey: ['logs'],
        queryFn: logsApi.get,
        refetchInterval: 5000
    })

    return (
        <div className="space-y-6">
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 flex flex-col h-[calc(100vh-200px)] overflow-hidden">
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-slate-100 bg-white z-10">
                    <div className="flex items-center gap-2">
                        <div className="bg-indigo-50 p-2 rounded-lg">
                            <ScrollText className="text-indigo-600" size={20} />
                        </div>
                        <div>
                            <h3 className="text-lg font-bold text-slate-800">System Logs</h3>
                            <p className="text-xs text-slate-500">
                                최근 {logsData?.logs.length ?? 0}개의 활동 로그
                            </p>
                        </div>
                    </div>

                    <button
                        onClick={() => { void refetch() }}
                        className="p-2 hover:bg-slate-100 rounded-lg text-slate-500 transition-colors"
                        title="Refresh Logs"
                    >
                        <RefreshCw size={18} className={isRefetching ? "animate-spin" : ""} />
                    </button>
                </div>

                {/* Log List */}
                <div className="flex-1 overflow-auto bg-slate-50 scrollbar-thin scrollbar-thumb-slate-200 scrollbar-track-transparent">
                    {isLoading ? (
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
                        <div className="bg-white">
                            <div className="hidden pid-header border-b border-slate-100 bg-slate-50 p-2 text-xs font-bold text-slate-500 uppercase tracking-wider md:grid md:grid-cols-[150px_140px_1fr] pl-3">
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

                {/* Footer */}
                <div className="p-3 border-t border-slate-100 bg-slate-50 text-[11px] text-slate-400 text-center shrink-0 font-medium">
                    Showing last 100 entries • Auto-refreshing every 5s
                </div>
            </div>
        </div>
    )
}

export default LogsTab
