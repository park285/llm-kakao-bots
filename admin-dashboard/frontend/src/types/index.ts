// API Response Types
export interface ApiResponse<T = unknown> {
  status: string
  message?: string
  error?: string
  data?: T
}

// Member Types
export interface Member {
  id: number
  channelId: string
  name: string
  aliases: {
    ko: string[]
    ja: string[]
  }
  nameJa?: string
  nameKo?: string
  isGraduated: boolean
}

export interface MembersResponse {
  status: string
  members: Member[]
}

// Alarm Types
export interface Alarm {
  roomId: string
  roomName: string
  userId: string
  userName: string
  channelId: string
  memberName: string
}

export interface AlarmsResponse {
  status: string
  alarms: Alarm[]
}

// Room Types
export interface RoomsResponse {
  status: string
  rooms: string[]
  aclEnabled: boolean
}

// Stats Types
export interface StatsResponse {
  status: string
  members: number
  alarms: number
  rooms: number
  version: string
  uptime: string
}

// Auth Types
export interface LoginCredentials {
  username: string
  password: string
}

export interface HeartbeatResponse {
  status?: string
  rotated?: boolean
  absolute_expires_at?: number
  idle_rejected?: boolean
  absolute_expired?: boolean
  error?: string
}

// Mutation Request Types
export interface AddAliasRequest {
  type: 'ko' | 'ja'
  alias: string
}

export interface RemoveAliasRequest {
  type: 'ko' | 'ja'
  alias: string
}

export interface SetGraduationRequest {
  isGraduated: boolean
}

export interface UpdateChannelRequest {
  channelId: string
}

export interface AddRoomRequest {
  room: string
}

export interface RemoveRoomRequest {
  room: string
}

export interface DeleteAlarmRequest {
  roomId: string
  userId: string
  channelId: string
}

// Stream Types
export interface Stream {
  id: string
  title: string
  status: string
  channel_name?: string
  channel_id: string
  link?: string
  thumbnail?: string
  start_scheduled?: string
  start_actual?: string
}

export interface StreamsResponse {
  status: string
  streams: Stream[]
}

// Channel Stats Types
export interface ChannelStat {
  ChannelID: string
  ChannelTitle: string
  SubscriberCount: number
  VideoCount: number
  ViewCount: number
}

export interface ChannelStatsResponse {
  status: string
  stats: Record<string, ChannelStat>
}

// Log Types
export interface LogEntry {
  timestamp: string
  type: string
  summary: string
  details?: Record<string, unknown>
}

export interface LogsResponse {
  status: string
  logs: LogEntry[]
}

// Settings Types
export interface Settings {
  alarmAdvanceMinutes: number
}

export interface SettingsResponse {
  status: string
  settings: Settings
}

export interface ServiceGoroutines {
  name: string
  goroutines: number
  available: boolean
}

export interface SystemStats {
  cpuUsage: number
  memoryUsage: number
  memoryTotal: number
  memoryUsed: number
  goroutines: number
  totalGoroutines: number
  serviceGoroutines: ServiceGoroutines[]
}

// Docker Types
export interface DockerContainer {
  name: string
  id: string
  image: string
  state: string
  status: string
  health: string
  managed: boolean
  paused: boolean
  startedAt?: string
}

// Milestone Types
export interface Milestone {
  channelId: string
  memberName: string
  type: string
  value: number
  achievedAt: string
  notified: boolean
}

export interface MilestonesResponse {
  status: string
  milestones: Milestone[]
  total: number
  limit: number
  offset: number
}

export interface NearMilestone {
  channelId: string
  memberName: string
  currentSubs: number
  nextMilestone: number
  remaining: number
  progressPct: number
}

export interface NearMilestonesResponse {
  status: string
  members: NearMilestone[]
  count: number
  threshold: number
}

export interface MilestoneStats {
  totalAchieved: number
  totalNearMilestone: number
  recentAchievements: number
  notNotifiedCount: number
}

export interface MilestoneStatsResponse {
  status: string
  stats: MilestoneStats
}

// Trace Types (Jaeger Integration)

export interface TraceSummary {
  traceId: string
  spanCount: number
  services: string[]
  operationName: string
  duration: number        // μs (microseconds)
  startTime: string       // ISO 8601 UTC
  hasError: boolean
}

export interface TraceSearchParams {
  service: string
  operation?: string
  limit?: number
  lookback?: string       // "1h", "6h", "24h", "7d"
  minDuration?: string    // "100ms", "1s"
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
  timestamp: number       // μs
  fields: Array<{ key: string; value: unknown }>
}

export interface Span {
  spanId: string
  traceId: string
  operationName: string
  serviceName: string     // Backend Enrichment로 주입됨
  duration: number        // μs
  startTime: number       // Unix μs
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

// Span Tree Node (Frontend 변환용)
export interface SpanNode extends Span {
  children: SpanNode[]
  depth: number
}

// Dependencies Types
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

// Metrics Types
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
