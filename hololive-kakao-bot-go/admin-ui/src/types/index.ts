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
