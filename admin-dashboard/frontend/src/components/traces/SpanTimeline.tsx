import { useRef, useMemo } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import type { SpanNode } from '@/types'
import { flattenTree } from '@/components/traces/utils'
import { SpanRow } from '@/components/traces/SpanRow'

interface SpanTimelineProps {
    spanTree: SpanNode[]
    traceStart: number
    traceDuration: number
}

/**
 * SpanTimeline: Span 트리를 가상화된 리스트로 렌더링함
 */
export const SpanTimeline = ({ spanTree, traceStart, traceDuration }: SpanTimelineProps) => {
    const parentRef = useRef<HTMLDivElement>(null)

    // 1. 트리 평탄화 (가상화를 위해)
    // 현재는 모든 노드를 펼친 상태로 간주함
    const flattenedSpans = useMemo(() => flattenTree(spanTree), [spanTree])

    // 2. 가상화 설정
    const rowVirtualizer = useVirtualizer({
        count: flattenedSpans.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 36, // 각 행의 대략적인 높이
        overscan: 5,
    })

    // 3. Grid Lines (배경 눈금)
    const gridLines = useMemo(() => {
        const lines = []
        const step = 10 // 10% 단위
        for (let i = 0; i <= 100; i += step) {
            lines.push(i)
        }
        return lines
    }, [])

    return (
        <div className="flex flex-col h-full min-h-[400px] border rounded-lg overflow-hidden bg-white">
            {/* Header */}
            <div className="flex h-9 border-b border-slate-200 bg-slate-50 text-xs font-semibold text-slate-500 uppercase tracking-wider shrink-0 z-10">
                <div className="w-[30%] min-w-[300px] px-3 flex items-center border-r border-slate-200">
                    Operation
                </div>
                <div className="flex-1 relative flex items-center px-2">
                    Timeline
                    {/* 눈금 라벨은 필요에 따라 추가함 */}
                </div>
            </div>

            {/* Virtualized Body */}
            <div
                ref={parentRef}
                className="flex-1 overflow-auto relative"
            >
                {/* Grid Background */}
                <div className="absolute inset-0 flex pointer-events-none h-full">
                    <div className="w-[30%] min-w-[300px] border-r border-slate-100 bg-slate-50/20" />
                    <div className="flex-1 relative h-full">
                        {gridLines.map((pos) => (
                            <div
                                key={pos}
                                className="absolute top-0 bottom-0 border-l border-slate-100 border-dashed first:border-solid h-full"
                                style={{ left: `${String(pos)}%` }}
                            />
                        ))}
                    </div>
                </div>

                <div
                    style={{
                        height: `${String(rowVirtualizer.getTotalSize())}px`,
                        width: '100%',
                        position: 'relative',
                    }}
                >
                    {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                        const span = flattenedSpans[virtualRow.index]
                        if (!span) return null

                        return (
                            <SpanRow
                                key={span.spanId}
                                span={span}
                                traceStart={traceStart}
                                traceDuration={traceDuration}
                                style={{
                                    position: 'absolute',
                                    top: 0,
                                    left: 0,
                                    width: '100%',
                                    height: `${String(virtualRow.size)}px`,
                                    transform: `translateY(${String(virtualRow.start)}px)`,
                                }}
                                hasChildren={span.children.length > 0}
                                // TODO: 접기/펼치기 기능은 상태 관리 복잡도 때문에 추후 구현함
                                isExpanded={true}
                            />
                        )
                    })}
                </div>
            </div>
        </div>
    )
}
