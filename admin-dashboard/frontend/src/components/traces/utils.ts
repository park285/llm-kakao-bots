import type { Span, SpanNode } from '@/types'

/**
 * buildSpanTree: Flat Span 배열을 Tree 구조로 변환함
 * Backend가 반환하는 평탄화된 리스트를 부모-자식 관계로 재구성함
 */
export function buildSpanTree(spans: Span[]): SpanNode[] {
    const spanMap = new Map<string, SpanNode>()
    const roots: SpanNode[] = []

    // 1. 모든 Span을 Map에 등록함
    spans.forEach(span => {
        spanMap.set(span.spanId, { ...span, children: [], depth: 0 })
    })

    // 2. 부모-자식 관계 구축함
    spans.forEach(span => {
        const node = spanMap.get(span.spanId)
        if (!node) return
        const parentRef = span.references.find(r => r.refType === 'CHILD_OF')

        // 부모가 존재하고 Map에 있다면 자식으로 추가함
        if (parentRef && spanMap.has(parentRef.spanId)) {
            const parent = spanMap.get(parentRef.spanId)
            if (parent) {
                node.depth = parent.depth + 1
                parent.children.push(node)
            }
        } else {
            // 부모가 없거나 찾을 수 없으면 루트로 간주함
            roots.push(node)
        }
    })

    // 3. 각 레벨에서 startTime 기준 정렬함
    const sortByStartTime = (nodes: SpanNode[]) => {
        nodes.sort((a, b) => a.startTime - b.startTime)
        nodes.forEach(node => {
            sortByStartTime(node.children)
        })
    }
    sortByStartTime(roots)

    return roots
}

/**
 * flattenTree: Tree를 평탄화 (가상 스크롤용)
 * DFS 순서로 노드 반환함
 */
export function flattenTree(roots: SpanNode[]): SpanNode[] {
    const result: SpanNode[] = []

    const dfs = (node: SpanNode) => {
        result.push(node)
        node.children.forEach(dfs)
    }

    roots.forEach(dfs)
    return result
}

/**
 * formatDuration: Duration 포맷팅 (μs -> 사람이 읽기 좋은 형식)
 */
export function formatDuration(microseconds: number): string {
    if (microseconds < 1000) {
        return `${String(microseconds)}μs`
    }
    if (microseconds < 1000000) {
        return `${(microseconds / 1000).toFixed(2)}ms`
    }
    return `${(microseconds / 1000000).toFixed(2)}s`
}

/**
 * getRelativePosition: 상대 시간 계산 (Trace 시작 기준)
 */
export function getRelativePosition(
    spanStart: number,
    traceStart: number,
    traceDuration: number
): number {
    if (traceDuration === 0) return 0
    return ((spanStart - traceStart) / traceDuration) * 100
}

/**
 * getSpanWidth: Span 너비 계산 (Trace Duration 기준 %)
 */
export function getSpanWidth(
    spanDuration: number,
    traceDuration: number
): number {
    if (traceDuration === 0) return 100
    return Math.max((spanDuration / traceDuration) * 100, 0.5) // 최소 0.5%
}

/**
 * getServiceColor: 서비스별 색상 생성 (일관된 해시 기반)
 */
const serviceColors = [
    'bg-sky-500',
    'bg-emerald-500',
    'bg-amber-500',
    'bg-rose-500',
    'bg-violet-500',
    'bg-cyan-500',
    'bg-pink-500',
    'bg-indigo-500',
]

/**
 * getServiceColor: 서비스 이름에 기반한 일관된 색상을 반환함
 */
export function getServiceColor(serviceName: string): string {
    let hash = 0
    for (let i = 0; i < serviceName.length; i++) {
        hash = serviceName.charCodeAt(i) + ((hash << 5) - hash)
    }
    const index = Math.abs(hash) % serviceColors.length
    return serviceColors[index] ?? 'bg-slate-500' // Fallback color
}
