import apiClient from './client'
import type {
  MembersResponse,
  AlarmsResponse,
  RoomsResponse,
  StatsResponse,
  AddAliasRequest,
  RemoveAliasRequest,
  SetGraduationRequest,
  UpdateChannelRequest,
  AddRoomRequest,
  RemoveRoomRequest,
  DeleteAlarmRequest,

  ApiResponse,
  StreamsResponse,
  ChannelStatsResponse,
  LogsResponse,
  SettingsResponse,
  Settings,
  Member,
  MilestonesResponse,
  NearMilestonesResponse,
  MilestoneStatsResponse,
  HeartbeatResponse,
} from '@/types'

// Auth API
export const authApi = {
  login: async (username: string, password: string): Promise<void> => {
    const response = await apiClient.post<{ success?: boolean; error?: string }>('/login', { username, password })
    // 서버가 항상 200을 반환하므로 본문의 success 필드 확인
    if (response.data.success === false) {
      throw new Error(response.data.error || 'Authentication failed')
    }
  },

  logout: async () => {
    await apiClient.get('/logout')
  },

  heartbeat: async (idle = false): Promise<HeartbeatResponse> => {
    try {
      const response = await apiClient.post<HeartbeatResponse>('/heartbeat', { idle })
      return response.data
    } catch (error: any) {
      if (error.response?.data) {
        return error.response.data as HeartbeatResponse
      }
      throw error
    }
  },
}

// Members API
export const membersApi = {
  getAll: async () => {
    const response = await apiClient.get<MembersResponse>('/members')
    return response.data
  },

  add: async (member: Partial<Member>) => {
    const response = await apiClient.post<ApiResponse>('/members', member)
    return response.data
  },

  addAlias: async (memberId: number, request: AddAliasRequest) => {
    const response = await apiClient.post<ApiResponse>(
      `/members/${String(memberId)}/aliases`,
      request
    )
    return response.data
  },

  removeAlias: async (memberId: number, request: RemoveAliasRequest) => {
    const response = await apiClient.delete<ApiResponse>(
      `/members/${String(memberId)}/aliases`,
      { data: request }
    )
    return response.data
  },

  setGraduation: async (memberId: number, request: SetGraduationRequest) => {
    const response = await apiClient.patch<ApiResponse>(
      `/members/${String(memberId)}/graduation`,
      request
    )
    return response.data
  },

  updateChannel: async (memberId: number, request: UpdateChannelRequest) => {
    const response = await apiClient.patch<ApiResponse>(
      `/members/${String(memberId)}/channel`,
      request
    )
    return response.data
  },

  updateName: async (memberId: number, name: string) => {
    const response = await apiClient.patch<ApiResponse>(
      `/members/${String(memberId)}/name`,
      { name }
    )
    return response.data
  },
}

// Alarms API
export const alarmsApi = {
  getAll: async () => {
    const response = await apiClient.get<AlarmsResponse>('/alarms')
    return response.data
  },

  delete: async (request: DeleteAlarmRequest) => {
    const response = await apiClient.delete<ApiResponse>('/alarms', {
      data: request,
    })
    return response.data
  },
}

// Rooms API
export const roomsApi = {
  getAll: async () => {
    const response = await apiClient.get<RoomsResponse>('/rooms')
    return response.data
  },

  add: async (request: AddRoomRequest) => {
    const response = await apiClient.post<ApiResponse>('/rooms', request)
    return response.data
  },

  remove: async (request: RemoveRoomRequest) => {
    const response = await apiClient.delete<ApiResponse>('/rooms', {
      data: request,
    })
    return response.data
  },

  setACL: async (enabled: boolean) => {
    const response = await apiClient.post<ApiResponse & { enabled: boolean }>('/rooms/acl', { enabled })
    return response.data
  },
}

// Stats API
export const statsApi = {
  get: async () => {
    const response = await apiClient.get<StatsResponse>('/stats')
    return response.data
  },
  getChannels: async () => {
    const response = await apiClient.get<ChannelStatsResponse>('/stats/channels')
    return response.data
  },
}

// Streams API
export const streamsApi = {
  getLive: async () => {
    const response = await apiClient.get<StreamsResponse>('/streams/live')
    return response.data
  },
  getUpcoming: async () => {
    const response = await apiClient.get<StreamsResponse>('/streams/upcoming')
    return response.data
  }
}

// Logs API
export const logsApi = {
  get: async () => {
    const response = await apiClient.get<LogsResponse>('/logs')
    return response.data
  }
}

// Settings API
export const settingsApi = {
  get: async () => {
    const response = await apiClient.get<SettingsResponse>('/settings')
    return response.data
  },
  update: async (settings: Settings) => {
    const response = await apiClient.post<ApiResponse>('/settings', settings)
    return response.data
  }
}

// Names API
export const namesApi = {
  setRoomName: async (roomId: string, roomName: string) => {
    const response = await apiClient.post<ApiResponse>('/names/room', {
      roomId,
      roomName,
    })
    return response.data
  },

  setUserName: async (userId: string, userName: string) => {
    const response = await apiClient.post<ApiResponse>('/names/user', {
      userId,
      userName,
    })
    return response.data
  },
}

// Docker API (컨테이너 관리)
// DockerContainer 타입은 @/types에서 import
import type { DockerContainer } from '@/types'
export type { DockerContainer }

export const dockerApi = {
  checkHealth: async () => {
    const response = await apiClient.get<{ status: string; available: boolean }>('/docker/health')
    return response.data
  },

  getContainers: async () => {
    const response = await apiClient.get<{ status: string; containers: DockerContainer[] }>('/docker/containers')
    return response.data
  },

  restartContainer: async (name: string) => {
    const response = await apiClient.post<ApiResponse>(`/docker/containers/${name}/restart`)
    return response.data
  },

  stopContainer: async (name: string) => {
    const response = await apiClient.post<ApiResponse>(`/docker/containers/${name}/stop`)
    return response.data
  },

  startContainer: async (name: string) => {
    const response = await apiClient.post<ApiResponse>(`/docker/containers/${name}/start`)
    return response.data
  },
}

// Milestones API
export interface GetMilestonesParams {
  limit?: number
  offset?: number
  channelId?: string
  memberName?: string
}

export const milestonesApi = {
  getAchieved: async (params?: GetMilestonesParams) => {
    const response = await apiClient.get<MilestonesResponse>('/milestones', {
      params: {
        limit: 50,
        ...params
      },
    })
    return response.data
  },

  getNear: async (threshold = 0.9) => {
    const response = await apiClient.get<NearMilestonesResponse>('/milestones/near', {
      params: { threshold },
    })
    return response.data
  },

  getStats: async () => {
    const response = await apiClient.get<MilestoneStatsResponse>('/milestones/stats')
    return response.data
  },
}
