import { useMemo, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { alarmsApi, namesApi } from '@/api'
import type { Alarm } from '@/types'
import EditNameModal from './EditNameModal'

interface AlarmGroup {
  roomId: string
  roomName: string
  userId: string
  userName: string
  alarms: Alarm[]
}

const AlarmsTab = () => {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())
  const [editModal, setEditModal] = useState<{
    type: 'room' | 'user'
    id: string
    currentName: string
  } | null>(null)

  const { data: response, isLoading } = useQuery({
    queryKey: ['alarms'],
    queryFn: alarmsApi.getAll,
  })

  const deleteAlarmMutation = useMutation({
    mutationFn: alarmsApi.delete,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] })
    },
  })

  const setNameMutation = useMutation({
    mutationFn: async ({ type, id, name }: { type: 'room' | 'user'; id: string; name: string }) => {
      if (type === 'room') {
        return namesApi.setRoomName(id, name)
      }
      return namesApi.setUserName(id, name)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] })
    },
  })

  // 유저별로 그룹화
  const groupedAlarms = useMemo(() => {
    const alarms = response?.alarms || []
    const groups = new Map<string, AlarmGroup>()

    alarms.forEach((alarm: Alarm) => {
      const key = `${alarm.roomId}:${alarm.userId}`
      if (!groups.has(key)) {
        groups.set(key, {
          roomId: alarm.roomId,
          roomName: alarm.roomName,
          userId: alarm.userId,
          userName: alarm.userName,
          alarms: [],
        })
      }
      const group = groups.get(key)
      if (group) {
        group.alarms.push(alarm)
      }
    })

    // 한글 이름 기준 정렬 (방 이름 → 유저 이름)
    return Array.from(groups.values()).sort((a, b) => {
      if (a.roomName !== b.roomName) return a.roomName.localeCompare(b.roomName, 'ko')
      return a.userName.localeCompare(b.userName, 'ko')
    })
  }, [response])

  // 검색 필터링
  const filteredGroups = useMemo(() => {
    if (!search.trim()) return groupedAlarms

    const searchLower = search.toLowerCase()
    return groupedAlarms.filter(group =>
      group.roomName.toLowerCase().includes(searchLower) ||
      group.userName.toLowerCase().includes(searchLower) ||
      group.alarms.some(a => a.memberName.toLowerCase().includes(searchLower))
    )
  }, [groupedAlarms, search])

  const toggleGroup = (key: string) => {
    const newExpanded = new Set(expandedGroups)
    if (newExpanded.has(key)) {
      newExpanded.delete(key)
    } else {
      newExpanded.add(key)
    }
    setExpandedGroups(newExpanded)
  }

  const totalAlarms = filteredGroups.reduce((sum, g) => sum + g.alarms.length, 0)

  const handleDelete = (alarm: Alarm) => {
    if (!confirm(`알람을 삭제하시겠습니까?
채널: ${alarm.memberName || alarm.channelId}`)) return

    void deleteAlarmMutation.mutateAsync({
      roomId: alarm.roomId,
      userId: alarm.userId,
      channelId: alarm.channelId,
    })
  }

  const handleEditName = (type: 'room' | 'user', id: string, currentName: string) => {
    setEditModal({ type, id, currentName })
  }

  const handleSaveName = (newName: string) => {
    if (!editModal) return
    void setNameMutation.mutateAsync({
      type: editModal.type,
      id: editModal.id,
      name: newName,
    })
  }

  if (isLoading) {
    return <div className="text-center py-8 text-gray-600">로딩 중...</div>
  }

  if (groupedAlarms.length === 0) {
    return <div className="text-center py-8 text-gray-500">등록된 알람이 없습니다</div>
  }

  return (
    <div className="space-y-4">
      {/* 검색 바 */}
      <div className="bg-white border border-gray-200 rounded-lg p-4">
        <div className="flex items-center gap-3">
          <span className="text-2xl">🔍</span>
          <input
            type="text"
            value={search}
            onChange={(e) => { setSearch(e.target.value) }}
            placeholder="방 이름, 유저 이름, 멤버 이름으로 검색..."
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          {search && (
            <button
              onClick={() => { setSearch('') }}
              className="px-3 py-2 text-gray-600 hover:text-gray-900"
            >
              ✕
            </button>
          )}
        </div>
        <div className="mt-2 text-sm text-gray-600">
          총 {filteredGroups.length}개 그룹, {totalAlarms}개 알람
        </div>
      </div>

      {/* 알람 그룹 목록 */}
      <div className="space-y-3">
        {filteredGroups.length === 0 ? (
          <div className="text-center py-8 text-gray-500">검색 결과가 없습니다</div>
        ) : (
          filteredGroups.map((group) => {
            const groupKey = `${group.roomId}:${group.userId}`
            const isExpanded = expandedGroups.has(groupKey)
            const displayAlarms = isExpanded ? group.alarms : group.alarms.slice(0, 5)
            const hasMore = group.alarms.length > 5

            return (
              <div key={groupKey} className="bg-white border border-gray-200 rounded-lg overflow-hidden">
                {/* 그룹 헤더 (클릭하여 펼침/접기) */}
                <div
                  onClick={() => { toggleGroup(groupKey) }}
                  className="bg-gradient-to-r from-blue-50 to-indigo-50 px-4 py-4 border-b border-gray-200 cursor-pointer hover:from-blue-100 hover:to-indigo-100 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-semibold text-gray-900 flex items-center gap-2">
                        <span>📍</span>
                        <span>{group.roomName}</span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            handleEditName('room', group.roomId, group.roomName)
                          }}
                          className="text-sm text-blue-600 hover:text-blue-800"
                          title="방 이름 편집"
                        >
                          ✏️
                        </button>
                        <span className="text-gray-400">/</span>
                        <span>👤</span>
                        <span>{group.userName}</span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            handleEditName('user', group.userId, group.userName)
                          }}
                          className="text-sm text-blue-600 hover:text-blue-800"
                          title="유저 이름 편집"
                        >
                          ✏️
                        </button>
                      </div>
                      <div className="text-sm text-gray-600 mt-1">
                        알람 {group.alarms.length}개
                        {!isExpanded && hasMore && ` (${String(displayAlarms.length)}개 표시 중)`}
                      </div>
                    </div>
                    <button className="text-2xl text-gray-400 hover:text-gray-600">
                      {isExpanded ? '▲' : '▼'}
                    </button>
                  </div>
                </div>

                {/* 알람 목록 */}
                <div className="divide-y divide-gray-200">
                  {displayAlarms.map((alarm: Alarm, index: number) => (
                    <div key={`${alarm.channelId}-${String(index)}`} className="px-4 py-3 hover:bg-gray-50 flex items-center justify-between">
                      <div className="flex-1">
                        <div className="font-medium text-gray-900">
                          🎨 {alarm.memberName || '이름 없음'}
                        </div>
                        <div className="text-xs text-gray-400 mt-1 font-mono">
                          {alarm.channelId}
                        </div>
                      </div>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDelete(alarm)
                        }}
                        disabled={deleteAlarmMutation.isPending}
                        className="px-3 py-1.5 text-sm bg-red-600 text-white rounded hover:bg-red-700 disabled:bg-gray-400 transition-colors"
                      >
                        삭제
                      </button>
                    </div>
                  ))}
                </div>

                {/* 더보기 버튼 */}
                {!isExpanded && hasMore && (
                  <div className="bg-gray-50 px-4 py-3 text-center border-t border-gray-200">
                    <button
                      onClick={() => { toggleGroup(groupKey) }}
                      className="text-sm text-blue-600 hover:text-blue-800 font-medium"
                    >
                      더보기 ({String(group.alarms.length - displayAlarms.length)}개)
                    </button>
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>

      {/* 이름 편집 모달 */}
      <EditNameModal
        isOpen={editModal !== null}
        onClose={() => { setEditModal(null) }}
        type={editModal?.type || 'room'}
        id={editModal?.id || ''}
        currentName={editModal?.currentName || ''}
        onSave={handleSaveName}
      />
    </div>
  )
}

export default AlarmsTab
