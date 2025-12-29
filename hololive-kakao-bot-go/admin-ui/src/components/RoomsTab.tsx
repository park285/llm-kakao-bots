import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { roomsApi } from '@/api'

const RoomsTab = () => {
  const queryClient = useQueryClient()
  const [newRoom, setNewRoom] = useState('')

  const { data: response, isLoading } = useQuery({
    queryKey: ['rooms'],
    queryFn: roomsApi.getAll,
  })

  const addRoomMutation = useMutation({
    mutationFn: roomsApi.add,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['rooms'] })
      setNewRoom('')
    },
  })

  const removeRoomMutation = useMutation({
    mutationFn: roomsApi.remove,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['rooms'] })
    },
  })

  const setACLMutation = useMutation({
    mutationFn: roomsApi.setACL,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['rooms'] })
    },
  })

  const handleAddRoom = () => {
    const room = newRoom.trim()
    if (!room) return
    void addRoomMutation.mutateAsync({ room })
  }

  const handleRemoveRoom = (room: string) => {
    if (!confirm(`방을 삭제하시겠습니까?\n${room}`)) return
    void removeRoomMutation.mutateAsync({ room })
  }

  const handleToggleACL = () => {
    const newValue = !response?.aclEnabled
    void setACLMutation.mutateAsync(newValue)
  }

  if (isLoading) {
    return <div className="text-center py-8 text-gray-600">로딩 중...</div>
  }

  const rooms = response?.rooms || []
  const aclEnabled = response?.aclEnabled ?? true

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6">
      {/* ACL 토글 섹션 */}
      <div className="flex items-center justify-between mb-6 pb-4 border-b border-gray-200">
        <div>
          <h3 className="text-lg font-semibold">방 접근 제어 (ACL)</h3>
          <p className="text-sm text-gray-500 mt-1">
            {aclEnabled
              ? '활성화됨: 아래 화이트리스트에 등록된 방에서만 봇이 응답합니다.'
              : '비활성화됨: 모든 방에서 봇이 응답합니다.'
            }
          </p>
        </div>
        <button
          onClick={handleToggleACL}
          disabled={setACLMutation.isPending}
          className={`relative inline-flex h-8 w-14 items-center rounded-full transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-offset-2 ${aclEnabled
            ? 'bg-blue-600 focus:ring-blue-500'
            : 'bg-gray-300 focus:ring-gray-400'
            } ${setACLMutation.isPending ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
        >
          <span
            className={`inline-block h-6 w-6 transform rounded-full bg-white shadow-md transition-transform duration-200 ease-in-out ${aclEnabled ? 'translate-x-7' : 'translate-x-1'
              }`}
          />
        </button>
      </div>

      <h3 className="text-lg font-semibold mb-4">허용된 카카오톡 방 목록</h3>

      {/* 방 목록 */}
      <div className={`space-y-2 mb-6 ${!aclEnabled ? 'opacity-50' : ''}`}>
        {rooms.length === 0 ? (
          <div className="text-gray-500 text-center py-4">등록된 방이 없습니다</div>
        ) : (
          rooms.map((room: string) => (
            <div key={room} className="flex items-center justify-between bg-gray-50 px-4 py-3 rounded-lg">
              <span className="font-medium text-gray-900">{room}</span>
              <button
                onClick={() => { handleRemoveRoom(room) }}
                disabled={removeRoomMutation.isPending}
                className="px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700 disabled:bg-gray-400"
              >
                삭제
              </button>
            </div>
          ))
        )}
      </div>

      {/* 새 방 추가 */}
      <div className="border-t border-gray-200 pt-6">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          새 방 추가
        </label>
        <p className="text-xs text-gray-500 mb-2">
          ⚠️ 채팅방 ID(숫자)를 입력하세요.
        </p>
        <div className="flex gap-2">
          <input
            type="text"
            value={newRoom}
            onChange={(e) => { setNewRoom(e.target.value) }}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAddRoom() }}
            className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder="채팅방 ID (예: 451788135895779)"
            disabled={addRoomMutation.isPending}
          />
          <button
            onClick={handleAddRoom}
            disabled={addRoomMutation.isPending || !newRoom.trim()}
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 transition-colors"
          >
            추가
          </button>
        </div>
      </div>
    </div>
  )
}

export default RoomsTab
