import { useState } from 'react'

interface ChannelEditModalProps {
  isOpen: boolean
  onClose: () => void
  onSave: (newChannelId: string) => void
  memberId: number
  memberName: string
  currentChannelId: string
}

const ChannelEditModal = ({
  isOpen,
  onClose,
  onSave,
  memberId,
  memberName,
  currentChannelId,
}: ChannelEditModalProps) => {
  const [channelId, setChannelId] = useState(currentChannelId)

  const handleSave = () => {
    const trimmed = channelId.trim()
    if (!trimmed || trimmed === currentChannelId) {
      onClose()
      return
    }
    onSave(trimmed)
    onClose()
  }

  // 모달이 열릴 때마다 현재 값으로 초기화
  if (isOpen && channelId !== currentChannelId) {
    setChannelId(currentChannelId)
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/40 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        {/* 헤더 */}
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">
            유튜브 채널 ID 수정
          </h3>
        </div>

        {/* 본문 */}
        <div className="px-6 py-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              멤버
            </label>
            <div className="px-3 py-2 bg-gray-50 border border-gray-200 rounded text-sm text-gray-600">
              {memberName} (ID: {memberId})
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              현재 채널 ID
            </label>
            <div className="px-3 py-2 bg-gray-50 border border-gray-200 rounded text-sm text-gray-600 font-mono break-all">
              {currentChannelId}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              새 채널 ID
            </label>
            <input
              type="text"
              value={channelId}
              onChange={(e) => { setChannelId(e.target.value) }}
              onKeyDown={(e) => { if (e.key === 'Enter') handleSave() }}
              placeholder="UC..."
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
              autoFocus
            />
            <p className="mt-1 text-xs text-gray-500">
              💡 유튜브 채널 ID는 보통 UC로 시작합니다
            </p>
          </div>
        </div>

        {/* 푸터 */}
        <div className="px-6 py-4 bg-gray-50 border-t border-gray-200 rounded-b-lg flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            취소
          </button>
          <button
            onClick={handleSave}
            disabled={!channelId.trim() || channelId === currentChannelId}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 transition-colors"
          >
            저장
          </button>
        </div>
      </div>
    </div>
  )
}

export default ChannelEditModal
