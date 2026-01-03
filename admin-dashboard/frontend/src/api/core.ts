/**
 * Core Admin API (자동 생성 클라이언트 래퍼)
 *
 * 이 파일은 swagger-typescript-api로 생성된 클라이언트를 래핑하여
 * 기존 코드와의 호환성을 유지합니다.
 */

import { isAxiosError } from 'axios'
import { Auth } from '@/api/generated/Auth'
import { Docker } from '@/api/generated/Docker'
import { Logs } from '@/api/generated/Logs'
import { Traces } from '@/api/generated/Traces'
import type {
    InternalServerContainerInfo,
} from '@/api/generated/data-contracts'

// API 공통 설정
const API_CONFIG = {
    baseURL: '/admin/api',
    withCredentials: true,
}

// 싱글톤 인스턴스
const authClient = new Auth(API_CONFIG)
const dockerClient = new Docker(API_CONFIG)
const logsClient = new Logs(API_CONFIG)
const tracesClient = new Traces(API_CONFIG)

// 기존 인터페이스 유지를 위한 타입 변환
export interface HeartbeatResponse {
    status?: string
    rotated?: boolean
    absolute_expires_at?: number
    idle_rejected?: boolean
    absolute_expired?: boolean
    error?: string
}

// DockerContainer 타입 (기존 타입과 호환 유지)
export interface DockerContainer {
    id: string
    name: string
    state: string
    status: string
    // 기존 UI에서 사용하는 필드 (Backend에서 제공하지 않으면 기본값)
    image: string
    health: string
    managed: boolean
    paused: boolean
    startedAt?: string
    // 리소스 메트릭
    cpuPercent?: number
    memoryUsageMB?: number
    memoryLimitMB?: number
    memoryPercent?: number
    networkRxMB?: number
    networkTxMB?: number
    blockReadMB?: number
    blockWriteMB?: number
    goroutineCount?: number
}

// Trace 관련 타입 (더 상세한 버전 유지)
export interface TraceSummary {
    traceId: string
    spanCount: number
    services: string[]
    operationName: string
    duration: number
    startTime: string
    hasError: boolean
}

export interface TraceSearchParams {
    service: string
    operation?: string
    limit?: number
    lookback?: string
    minDuration?: string
    maxDuration?: string
    tags?: Record<string, string>
}

export interface TraceSearchResponse {
    status: string
    traces: TraceSummary[]
    total: number
    limit: number
}

export interface SpanReference {
    refType: 'CHILD_OF' | 'FOLLOWS_FROM'
    traceId: string
    spanId: string
}

export interface SpanTag {
    key: string
    type: string
    value: string | number | boolean
}

export interface SpanLog {
    timestamp: number
    fields: Array<{ key: string; value: unknown }>
}

export interface Span {
    spanId: string
    traceId: string
    operationName: string
    serviceName: string
    duration: number
    startTime: number
    references: SpanReference[]
    tags: SpanTag[]
    logs: SpanLog[]
    processId: string
    hasError: boolean
}

export interface TraceProcess {
    serviceName: string
    tags: SpanTag[]
}

export interface TraceDetailResponse {
    status: string
    traceId: string
    spans: Span[]
    processes: Record<string, TraceProcess>
}

export interface ServicesResponse {
    status: string
    services: string[]
}

export interface OperationsResponse {
    status: string
    service: string
    operations: string[]
}

export interface TracesHealthResponse {
    status: string
    available: boolean
}

export interface Dependency {
    parent: string
    child: string
    callCount: number
}

export interface DependenciesResponse {
    status: string
    dependencies: Dependency[]
    count: number
}

export interface MetricsParams {
    lookback?: string
    quantile?: string
    spanKind?: string
    step?: string
    ratePer?: string
    groupByOperation?: boolean
}

export interface MetricPoint {
    timestamp: number
    value: number
}

export interface ServiceMetrics {
    name: string
    callRate: number
    errorRate: number
    p50Latency: number
    p95Latency: number
    p99Latency: number
    avgDuration: number
}

export interface OperationMetrics {
    operation: string
    callRate: number
    errorRate: number
    p50Latency: number
    p95Latency: number
    p99Latency: number
    avgDuration: number
}

export interface ServiceMetricsResponse {
    status: string
    service: string
    metrics: ServiceMetrics
    operations?: OperationMetrics[]
    latencies?: MetricPoint[]
    calls?: MetricPoint[]
    errors?: MetricPoint[]
}

// Auth API: 기존 인터페이스 유지
export const authApi = {
    login: async (username: string, password: string): Promise<void> => {
        const response = await authClient.loginCreate({ username, password })
        // 서버가 항상 200을 반환하므로 본문의 status 필드 확인
        if (response.data.status !== 'ok') {
            throw new Error(response.data.message || 'Authentication failed')
        }
    },

    logout: async () => {
        await authClient.logoutCreate()
    },

    heartbeat: async (idle = false): Promise<HeartbeatResponse> => {
        try {
            const response = await authClient.heartbeatCreate({ idle })
            return response.data as unknown as HeartbeatResponse
        } catch (error) {
            if (isAxiosError(error) && error.response?.data) {
                return error.response.data as HeartbeatResponse
            }
            throw error
        }
    },
}

// Docker API
export const dockerApi = {
    checkHealth: async () => {
        const response = await dockerClient.healthList()
        return response.data
    },

    getContainers: async () => {
        const response = await dockerClient.containersList()
        // 타입 변환: InternalServerContainerInfo -> DockerContainer
        const containers: DockerContainer[] = (response.data.containers ?? []).map(
            (c: InternalServerContainerInfo) => ({
                id: c.id ?? '',
                name: c.name ?? '',
                state: c.state ?? '',
                status: c.status ?? '',
                // 기본값 (Backend에서 제공하지 않는 필드)
                image: '',
                health: c.state === 'running' ? 'healthy' : 'unhealthy',
                managed: true,
                paused: false,
                // 리소스 메트릭
                cpuPercent: c.cpuPercent,
                memoryUsageMB: c.memoryUsageMB,
                memoryLimitMB: c.memoryLimitMB,
                memoryPercent: c.memoryPercent,
                networkRxMB: c.networkRxMB,
                networkTxMB: c.networkTxMB,
                blockReadMB: c.blockReadMB,
                blockWriteMB: c.blockWriteMB,
                goroutineCount: c.goroutineCount,
            }),
        )
        return { status: response.data.status ?? 'ok', containers }
    },

    restartContainer: async (name: string) => {
        const response = await dockerClient.containersRestartCreate(name)
        return { status: response.data.status ?? 'ok', message: response.data.message }
    },

    stopContainer: async (name: string) => {
        const response = await dockerClient.containersStopCreate(name)
        return { status: response.data.status ?? 'ok', message: response.data.message }
    },

    startContainer: async (name: string) => {
        const response = await dockerClient.containersStartCreate(name)
        return { status: response.data.status ?? 'ok', message: response.data.message }
    },
}

// System Logs API (Core)
export const systemLogsApi = {
    getSystemLogs: async (file = 'combined', lines = 200) => {
        const response = await logsClient.logsList({ file, lines })
        return {
            status: response.data.status ?? 'ok',
            file: response.data.file ?? file,
            lines: response.data.lines ?? [],
            count: response.data.count ?? 0,
            error: undefined as string | undefined, // 오류 메시지 (선택적)
        }
    },

    getSystemLogFiles: async () => {
        const response = await logsClient.filesList()
        return {
            status: response.data.status ?? 'ok',
            files: (response.data.files ?? []).map((f) => ({
                key: f.key ?? '',
                name: f.name ?? '',
                description: f.description ?? '',
                exists: true,
            })),
        }
    },
}

// Traces API
export const tracesApi = {
    checkHealth: async (): Promise<TracesHealthResponse> => {
        const response = await tracesClient.healthList()
        return {
            status: response.data.status ?? 'ok',
            available: response.data.available ?? false,
        }
    },

    getServices: async (): Promise<ServicesResponse> => {
        const response = await tracesClient.servicesList()
        return {
            status: response.data.status ?? 'ok',
            services: response.data.services ?? [],
        }
    },

    getOperations: async (service: string): Promise<OperationsResponse> => {
        const response = await tracesClient.operationsDetail(service)
        return {
            status: response.data.status ?? 'ok',
            service: response.data.service ?? service,
            operations: response.data.operations ?? [],
        }
    },

    search: async (params: TraceSearchParams): Promise<TraceSearchResponse> => {
        // 생성된 클라이언트가 tags를 지원하지 않아 직접 쿼리 구성
        const queryParams = new URLSearchParams()
        queryParams.set('service', params.service)
        if (params.operation) queryParams.set('operation', params.operation)
        if (params.lookback) queryParams.set('lookback', params.lookback)
        if (params.limit) queryParams.set('limit', String(params.limit))
        if (params.minDuration) queryParams.set('minDuration', params.minDuration)
        if (params.maxDuration) queryParams.set('maxDuration', params.maxDuration)
        // tags 파라미터 추가 (Jaeger v2: tag=key:value 형식)
        if (params.tags) {
            for (const [key, value] of Object.entries(params.tags)) {
                queryParams.append('tag', `${key}:${value}`)
            }
        }

        const response = await fetch(`/admin/api/traces?${queryParams.toString()}`, {
            credentials: 'include',
        })
        const data = await response.json()
        return {
            status: data.status ?? 'ok',
            traces: (data.traces ?? []) as TraceSummary[],
            total: data.total ?? 0,
            limit: data.limit ?? 20,
        }
    },

    getTrace: async (traceId: string): Promise<TraceDetailResponse> => {
        const response = await tracesClient.tracesDetail(traceId)
        return {
            status: response.data.status ?? 'ok',
            traceId: response.data.traceId ?? traceId,
            spans: (response.data.spans ?? []) as Span[],
            processes: (response.data.processes ?? {}) as Record<string, TraceProcess>,
        }
    },

    getDependencies: async (lookback = '24h'): Promise<DependenciesResponse> => {
        const response = await tracesClient.dependenciesList({ lookback })
        return {
            status: response.data.status ?? 'ok',
            dependencies: (response.data.dependencies ?? []) as Dependency[],
            count: response.data.count ?? 0,
        }
    },

    getMetrics: async (
        service: string,
        params: MetricsParams = {},
    ): Promise<ServiceMetricsResponse> => {
        const response = await tracesClient.metricsDetail(service, {
            lookback: params.lookback,
            quantile: params.quantile,
            spanKind: params.spanKind,
            step: params.step,
            ratePer: params.ratePer,
            groupByOperation: params.groupByOperation,
        })
        return {
            status: response.data.status ?? 'ok',
            service: response.data.service ?? service,
            metrics: (response.data.metrics ?? {}) as ServiceMetrics,
            operations: response.data.operations as OperationMetrics[] | undefined,
            latencies: response.data.latencies as MetricPoint[] | undefined,
            calls: response.data.calls as MetricPoint[] | undefined,
            errors: response.data.errors as MetricPoint[] | undefined,
        }
    },
}
