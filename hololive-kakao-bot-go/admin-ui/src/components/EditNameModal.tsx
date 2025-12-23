import { useState } from 'react'

interface EditNameModalProps {
  isOpen: boolean
  onClose: () => void
  type: 'room' | 'user'
  id: string
  currentName: string
  onSave: (newName: string) => void
}

const EditNameModal = ({ isOpen, onClose, type, id, currentName, onSave }: EditNameModalProps) => {
  const [name, setName] = useState(currentName)
  const isNumericId = /^\d+$/.test(currentName)

  const handleSave = () => {
    const trimmed = name.trim()
    if (!trimmed) return
    onSave(trimmed)
    onClose()
  }

  if (!isOpen) return null

  const label = type === 'room' ? '방 이름' : '유저 이름'
  const placeholder = type === 'room' ? '홀로라이브 알림방' : '카푸'

  return (
    <div className="fixed inset-0 bg-black/40 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        {/* 헤더 */}
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">
            {label} 설정
          </h3>
        </div>

        {/* 본문 */}
        <div className="px-6 py-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              ID
            </label>
            <div className="px-3 py-2 bg-gray-50 border border-gray-200 rounded text-sm text-gray-600 font-mono">
              {id}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {label}
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => { setName(e.target.value) }}
              onKeyDown={(e) => { if (e.key === 'Enter') handleSave() }}
              placeholder={placeholder}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              autoFocus
            />
            {isNumericId && (
              <p className="mt-1 text-xs text-amber-600">
                💡 현재 숫자 ID로 표시되고 있습니다. 한글 이름을 입력하세요.
              </p>
            )}
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
            disabled={!name.trim()}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 transition-colors"
          >
            저장
          </button>
        </div>
      </div>
    </div>
  )
}

export default EditNameModal
