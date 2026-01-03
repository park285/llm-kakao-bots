import { ChevronRight, AlertTriangle } from 'lucide-react'
import clsx from 'clsx'
import type { SpanNode } from '@/types'
import { getRelativePosition, getSpanWidth, getServiceColor, formatDuration } from '@/components/traces/utils'
import type { CSSProperties } from 'react'

interface SpanRowProps {
    span: SpanNode
    style: CSSProperties
    traceStart: number
    traceDuration: number
    isExpanded?: boolean
    onToggle?: () => void
    hasChildren?: boolean
}

/**
 * SpanRow: 개별 Span을 타임라인 바 형태의 행으로 렌더링함
 */
export const SpanRow = ({
    span,
    style,
    traceStart,
    traceDuration,
    isExpanded = true,
    onToggle,
    hasChildren = false,
}: SpanRowProps) => {
    const left = getRelativePosition(span.startTime, traceStart, traceDuration)
    const width = getSpanWidth(span.duration, traceDuration)
    const serviceColor = getServiceColor(span.serviceName)

    return (
        <div
            style={style}
            className={clsx(
                "flex items-center text-xs border-b border-slate-100 hover:bg-slate-50 transition-colors",
                span.hasError && "bg-rose-50/30 hover:bg-rose-50/50"
            )}
        >
            {/* 1. Operation Name & Hierarchy (Left Pane) */}
            <div className="w-[30%] min-w-[300px] border-r border-slate-100 h-full flex items-center px-2 overflow-hidden bg-white/50">
                <div
                    className="flex items-center min-w-0"
                    style={{ paddingLeft: `${String(span.depth * 16)}px` }}
                >
                    {/* Toggle Button for Tree */}
                    {hasChildren ? (
                        <button
                            onClick={onToggle}
                            className="p-0.5 rounded hover:bg-slate-200 mr-1 shrink-0"
                        >
                            <ChevronRight
                                size={12}
                                className={clsx(
                                    "text-slate-400 transition-transform",
                                    isExpanded && "rotate-90"
                                )}
                            />
                        </button>
                    ) : (
                        <div className="w-4 mr-1 shrink-0" />
                    )}

                    {/* Service Badge */}
                    <span
                        className={clsx(
                            "px-1.5 py-0.5 rounded text-[10px] text-white font-medium mr-2 shrink-0 truncate max-w-[80px]",
                            serviceColor
                        )}
                        title={span.serviceName}
                    >
                        {span.serviceName}
                    </span>

                    {/* Operation Name */}
                    <span
                        className={clsx(
                            "truncate font-mono",
                            span.hasError ? "text-rose-600 font-medium" : "text-slate-700"
                        )}
                        title={span.operationName}
                    >
                        {span.operationName}
                    </span>

                    {span.hasError && (
                        <AlertTriangle size={12} className="text-rose-500 ml-2 shrink-0" />
                    )}
                </div>
            </div>

            {/* 2. Timeline Bar (Right Pane) */}
            <div className="flex-1 h-full relative min-w-0">
                <div
                    className={clsx(
                        "absolute top-1.5 h-6 rounded min-w-[2px] shadow-sm transition-all opacity-90",
                        span.hasError ? "bg-rose-400" : serviceColor.replace("bg-", "bg-opacity-80 bg-")
                    )}
                    style={{
                        left: `${String(left)}%`,
                        width: `${String(width)}%`,
                    }}
                >
                    {/* Duration Label on Hover/Always */}
                    <span className="absolute -top-8 left-0 hidden group-hover:block bg-slate-800 text-white text-[10px] px-1.5 py-0.5 rounded z-10 whitespace-nowrap">
                        {formatDuration(span.duration)}
                    </span>

                    {/* Duration Text (Inside Bar if wide enough and near edge) */}
                    {left + width > 85 && width > 10 && (
                        <span className="absolute right-1 top-[3px] text-[10px] text-white font-medium whitespace-nowrap drop-shadow-sm">
                            {formatDuration(span.duration)}
                        </span>
                    )}
                </div>

                {/* Duration Text (Outside - Right aligned if normal, Left aligned if near edge but narrow) */}
                {!(left + width > 85 && width > 10) && (
                    <span
                        className={clsx(
                            "absolute text-[10px] text-slate-500 top-2.5 whitespace-nowrap",
                            left + width > 85 ? "-translate-x-full -ml-1.5" : "ml-1.5"
                        )}
                        style={{
                            left: left + width > 85
                                ? `${String(left)}%`
                                : `${String(left + width)}%`
                        }}
                    >
                        {formatDuration(span.duration)}
                    </span>
                )}
            </div>
        </div>
    )
}
