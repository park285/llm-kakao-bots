import { useState, useOptimistic } from 'react'
import ConfirmModal from './ConfirmModal'
import ChannelEditModal from './ChannelEditModal'
import AddMemberModal from './AddMemberModal'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { membersApi } from '@/api'
import type { Member } from '@/types'
import { Button, Badge, Card, Input } from '@/components/ui'
import { Search, GraduationCap, Edit2, Plus, ExternalLink, RotateCcw } from 'lucide-react'

const MembersTab = () => {
  const queryClient = useQueryClient()

  const { data: response, isLoading } = useQuery({
    queryKey: ['members'],
    queryFn: membersApi.getAll,
  })

  // Mutations (unchanged logic)
  const addAliasMutation = useMutation({
    mutationFn: ({ memberId, type, alias }: { memberId: number; type: 'ko' | 'ja'; alias: string }) =>
      membersApi.addAlias(memberId, { type, alias }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['members'] })
    },
  })

  const removeAliasMutation = useMutation({
    mutationFn: ({ memberId, type, alias }: { memberId: number; type: 'ko' | 'ja'; alias: string }) =>
      membersApi.removeAlias(memberId, { type, alias }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['members'] })
    },
  })

  const updateChannelMutation = useMutation({
    mutationFn: ({ memberId, channelId }: { memberId: number; channelId: string }) =>
      membersApi.updateChannel(memberId, { channelId }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['members'] })
    },
  })

  const setGraduationMutation = useMutation({
    mutationFn: ({ memberId, isGraduated }: { memberId: number; isGraduated: boolean }) =>
      membersApi.setGraduation(memberId, { isGraduated }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['members'] })
    },
  })

  const addMemberMutation = useMutation({
    mutationFn: membersApi.add,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['members'] })
    },
    onError: (err) => {
      alert(`Failed to add member: ${(err).message}`)
    }
  })

  // State management
  const [inputs, setInputs] = useState<Record<string, string>>({})
  const [searchTerm, setSearchTerm] = useState('')
  const [hideGraduated, setHideGraduated] = useState<boolean>(() => {
    const saved = localStorage.getItem('hideGraduated')
    return saved !== null ? saved === 'true' : true
  })

  // Modal state
  type ModalState =
    | { type: 'none' }
    | { type: 'removeAlias'; memberId: number; aliasType: 'ko' | 'ja'; alias: string }
    | { type: 'graduation'; memberId: number; memberName: string; currentStatus: boolean }
    | { type: 'channelEdit'; memberId: number; memberName: string; currentChannelId: string }

  const [modal, setModal] = useState<ModalState>({ type: 'none' })
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)

  // Optimistic updates setup (unchanged logic)
  const allMembers = (response?.members ?? []).map(m => ({
    ...m,
    aliases: {
      ko: m.aliases.ko,
      ja: m.aliases.ja,
    },
  }))

  type OptimisticUpdate =
    | { type: 'graduation'; memberId: number; isGraduated: boolean }
    | { type: 'addAlias'; memberId: number; aliasType: 'ko' | 'ja'; alias: string }
    | { type: 'removeAlias'; memberId: number; aliasType: 'ko' | 'ja'; alias: string }
    | { type: 'updateChannel'; memberId: number; channelId: string }

  const [optimisticMembers, setOptimisticMembers] = useOptimistic(
    allMembers,
    (state, update: OptimisticUpdate) => {
      switch (update.type) {
        case 'graduation':
          return state.map((m) =>
            m.id === update.memberId ? { ...m, isGraduated: update.isGraduated } : m
          )
        case 'addAlias':
          return state.map((m) =>
            m.id === update.memberId
              ? {
                ...m,
                aliases: {
                  ...m.aliases,
                  [update.aliasType]: [...m.aliases[update.aliasType], update.alias],
                },
              }
              : m
          )
        case 'removeAlias':
          return state.map((m) =>
            m.id === update.memberId
              ? {
                ...m,
                aliases: {
                  ...m.aliases,
                  [update.aliasType]: m.aliases[update.aliasType].filter(
                    (a) => a !== update.alias
                  ),
                },
              }
              : m
          )
        case 'updateChannel':
          return state.map((m) =>
            m.id === update.memberId ? { ...m, channelId: update.channelId } : m
          )
      }
    }
  )

  const toggleHideGraduated = () => {
    const newValue = !hideGraduated
    setHideGraduated(newValue)
    localStorage.setItem('hideGraduated', String(newValue))
  }

  const handleAddAlias = (memberId: number, type: 'ko' | 'ja') => {
    const key = `${String(memberId)}-${type}`
    const alias = inputs[key]?.trim()
    if (!alias) return

    setOptimisticMembers({ type: 'addAlias', memberId, aliasType: type, alias })
    void addAliasMutation.mutateAsync({ memberId, type, alias })
    setInputs({ ...inputs, [key]: '' })
  }

  const handleRemoveAlias = (memberId: number, type: 'ko' | 'ja', alias: string) => {
    setModal({ type: 'removeAlias', memberId, aliasType: type, alias })
  }

  const confirmRemoveAlias = () => {
    if (modal.type !== 'removeAlias') return

    setOptimisticMembers({ type: 'removeAlias', memberId: modal.memberId, aliasType: modal.aliasType, alias: modal.alias })
    void removeAliasMutation.mutateAsync({ memberId: modal.memberId, type: modal.aliasType, alias: modal.alias })
  }

  const handleUpdateChannel = (memberId: number, memberName: string, currentChannelId: string) => {
    setModal({ type: 'channelEdit', memberId, memberName, currentChannelId })
  }

  const confirmUpdateChannel = (newChannelId: string) => {
    if (modal.type !== 'channelEdit') return

    setOptimisticMembers({ type: 'updateChannel', memberId: modal.memberId, channelId: newChannelId })
    void updateChannelMutation.mutateAsync({ memberId: modal.memberId, channelId: newChannelId })
  }

  const handleToggleGraduation = (memberId: number, memberName: string, currentStatus: boolean) => {
    setModal({ type: 'graduation', memberId, memberName, currentStatus })
  }

  const confirmToggleGraduation = () => {
    if (modal.type !== 'graduation') return

    const newStatus = !modal.currentStatus
    setOptimisticMembers({ type: 'graduation', memberId: modal.memberId, isGraduated: newStatus })
    void setGraduationMutation.mutateAsync({ memberId: modal.memberId, isGraduated: newStatus })
  }

  if (isLoading) {
    return <div className="text-center py-12 text-slate-500">데이터를 불러오는 중입니다...</div>
  }

  const filteredMembers = optimisticMembers.filter((m: Member) => {
    if (hideGraduated && m.isGraduated) return false
    if (searchTerm) {
      const lowerSearch = searchTerm.toLowerCase()
      // 이름, 별명, ID, 채널ID 검색
      return (
        m.name.toLowerCase().includes(lowerSearch) ||
        m.channelId.toLowerCase().includes(lowerSearch) ||
        String(m.id).includes(lowerSearch) ||
        m.aliases.ko.some(a => a.toLowerCase().includes(lowerSearch)) ||
        m.aliases.ja.some(a => a.toLowerCase().includes(lowerSearch))
      )
    }
    return true
  })

  const sortedMembers = [...filteredMembers].sort((a: Member, b: Member) => {
    if (a.isGraduated !== b.isGraduated) {
      return a.isGraduated ? 1 : -1
    }
    return a.name.localeCompare(b.name)
  })

  return (
    <div className="space-y-6">
      {/* 필터 및 검색 바 */}
      <div className="flex flex-col md:flex-row gap-4 items-center justify-between bg-white p-4 rounded-2xl shadow-sm border border-slate-100">
        <div className="flex items-center gap-4 w-full md:w-auto">
          <label className="flex items-center gap-2 cursor-pointer bg-slate-50 px-3 py-2 rounded-lg hover:bg-slate-100 transition-colors">
            <input
              type="checkbox"
              checked={hideGraduated}
              onChange={toggleHideGraduated}
              className="w-4 h-4 text-sky-600 rounded focus:ring-sky-500 border-gray-300"
            />
            <span className="text-sm font-medium text-slate-700 select-none">졸업 멤버 숨기기</span>
          </label>
          <div className="text-xs text-slate-400 font-medium bg-slate-50 px-3 py-2 rounded-lg">
            <span className="text-slate-900 font-bold">{sortedMembers.length}</span> / {allMembers.length} 명
          </div>
        </div>
      </div>

      <div className="flex flex-col md:flex-row gap-4 items-center justify-between">
        <Button onClick={() => { setIsAddModalOpen(true); }} className="gap-2 shrink-0 bg-sky-500 hover:bg-sky-600 text-white text-sm font-bold shadow-sm shadow-sky-200">
          <Plus size={16} /> 멤버 추가
        </Button>

        <div className="relative w-full md:w-80">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-slate-400">
            <Search size={16} />
          </div>
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => { setSearchTerm(e.target.value); }}
            placeholder="멤버 이름, ID, 별명 검색..."
            className="block w-full pl-10 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-sky-500/20 focus:border-sky-500 transition-all placeholder:text-slate-400"
          />
        </div>
      </div>


      {/* 멤버 카드 그리드 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
        {sortedMembers.map((member: Member) => (
          <Card key={member.id} className="relative group overflow-hidden border-slate-200">
            <Card.Header className="pb-3 border-b border-slate-50">
              <div className="flex items-start justify-between">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs font-mono text-slate-400">#{String(member.id).padStart(3, '0')}</span>
                    {member.isGraduated && (
                      <Badge color="gray" className="text-[10px] px-1.5 py-0.5 shadow-none ring-1 ring-slate-200">
                        Graduated
                      </Badge>
                    )}
                  </div>
                  <h3 className="font-bold text-lg text-slate-800 leading-tight">{member.name}</h3>
                </div>

                <button
                  onClick={() => { handleToggleGraduation(member.id, member.name, member.isGraduated) }}
                  className={`p-2 rounded-lg transition-all ${member.isGraduated
                    ? 'text-slate-400 hover:text-emerald-600 hover:bg-emerald-50'
                    : 'text-slate-300 hover:text-rose-600 hover:bg-rose-50'
                    }`}
                  title={member.isGraduated ? '졸업 해제 (복귀)' : '졸업 처리'}
                >
                  {member.isGraduated ? <RotateCcw size={18} /> : <GraduationCap size={18} />}
                </button>
              </div>

              <div className="mt-3 flex items-center gap-2 text-xs text-slate-500 bg-slate-50 p-2 rounded-lg">
                <span className="truncate flex-1 font-mono">{member.channelId}</span>
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    handleUpdateChannel(member.id, member.name, member.channelId)
                  }}
                  className="p-1 hover:bg-white rounded shadow-sm text-sky-600 transition-colors"
                  title="채널 ID 수정"
                >
                  <Edit2 size={12} />
                </button>
                <a
                  href={`https://youtube.com/channel/${member.channelId}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="p-1 hover:bg-white rounded shadow-sm text-slate-400 hover:text-red-500 transition-colors"
                  title="유튜브 채널 이동"
                >
                  <ExternalLink size={12} />
                </a>
              </div>
            </Card.Header>

            <Card.Body className="space-y-4 pt-2">
              {/* 한국어 별명 */}
              <div>
                <div className="text-[11px] font-bold text-slate-400 uppercase tracking-wider mb-2 flex items-center gap-1">
                  <span className="w-1.5 h-1.5 rounded-full bg-sky-400"></span>
                  Korean Aliases
                </div>
                <div className="flex flex-wrap gap-1.5 mb-2 min-h-[24px]">
                  {member.aliases.ko.map((alias: string) => (
                    <Badge
                      key={alias}
                      color="sky"
                      onRemove={() => { handleRemoveAlias(member.id, 'ko', alias) }}
                    >
                      {alias}
                    </Badge>
                  ))}
                  {member.aliases.ko.length === 0 && (
                    <span className="text-xs text-slate-300 italic">등록된 별명이 없습니다</span>
                  )}
                </div>
                <div className="flex gap-1.5">
                  <Input
                    value={inputs[`${String(member.id)}-ko`] || ''}
                    onChange={(e) => { setInputs({ ...inputs, [`${String(member.id)}-ko`]: e.target.value }) }}
                    placeholder="별명 추가"
                    className="flex-1 h-8 text-xs bg-slate-50 border-slate-200"
                    onKeyDown={(e) => { if (e.key === 'Enter') handleAddAlias(member.id, 'ko') }}
                  />
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => { handleAddAlias(member.id, 'ko') }}
                    className="h-8 w-8 p-0 flex items-center justify-center bg-sky-500 hover:bg-sky-600"
                  >
                    <Plus size={14} />
                  </Button>
                </div>
              </div>

              {/* 일본어 별명 */}
              <div>
                <div className="text-[11px] font-bold text-slate-400 uppercase tracking-wider mb-2 flex items-center gap-1">
                  <span className="w-1.5 h-1.5 rounded-full bg-rose-400"></span>
                  Japanese Aliases
                </div>
                <div className="flex flex-wrap gap-1.5 mb-2 min-h-[24px]">
                  {member.aliases.ja.map((alias: string) => (
                    <Badge
                      key={alias}
                      color="rose"
                      onRemove={() => { handleRemoveAlias(member.id, 'ja', alias) }}
                    >
                      {alias}
                    </Badge>
                  ))}
                  {member.aliases.ja.length === 0 && (
                    <span className="text-xs text-slate-300 italic">등록된 별명이 없습니다</span>
                  )}
                </div>
                <div className="flex gap-1.5">
                  <Input
                    value={inputs[`${String(member.id)}-ja`] || ''}
                    onChange={(e) => { setInputs({ ...inputs, [`${String(member.id)}-ja`]: e.target.value }) }}
                    placeholder="エイリアス追加"
                    className="flex-1 h-8 text-xs bg-slate-50 border-slate-200"
                    onKeyDown={(e) => { if (e.key === 'Enter') handleAddAlias(member.id, 'ja') }}
                  />
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => { handleAddAlias(member.id, 'ja') }}
                    className="h-8 w-8 p-0 flex items-center justify-center bg-rose-500 hover:bg-rose-600"
                  >
                    <Plus size={14} />
                  </Button>
                </div>
              </div>
            </Card.Body>
          </Card>
        ))}
      </div>

      {/* 별명 삭제 확인 모달 */}
      <ConfirmModal
        isOpen={modal.type === 'removeAlias'}
        onClose={() => { setModal({ type: 'none' }) }}
        onConfirm={confirmRemoveAlias}
        title="별명 삭제"
        message={modal.type === 'removeAlias' ? `정말 삭제하시겠습니까?` : ''}
        confirmText="삭제"
        confirmColor="red"
      >
        {modal.type === 'removeAlias' && (
          <div className="mt-2 p-3 bg-slate-50 rounded-lg text-center font-bold text-slate-700">
            {modal.alias}
          </div>
        )}
      </ConfirmModal>

      {/* 졸업 토글 확인 모달 */}
      <ConfirmModal
        isOpen={modal.type === 'graduation'}
        onClose={() => { setModal({ type: 'none' }) }}
        onConfirm={confirmToggleGraduation}
        title={modal.type === 'graduation' ? (modal.currentStatus ? '졸업 해제 (복귀)' : '졸업 처리') : ''}
        message={modal.type === 'graduation' ? `${modal.memberName}을(를) ${modal.currentStatus ? '졸업 해제' : '졸업 처리'}하시겠습니까?` : ''}
        confirmText="확인"
        confirmColor={modal.type === 'graduation' && modal.currentStatus ? 'blue' : 'red'}
      />

      {/* 채널 ID 수정 모달 */}
      <ChannelEditModal
        isOpen={modal.type === 'channelEdit'}
        onClose={() => { setModal({ type: 'none' }) }}
        onSave={confirmUpdateChannel}
        memberId={modal.type === 'channelEdit' ? modal.memberId : 0}
        memberName={modal.type === 'channelEdit' ? modal.memberName : ''}
        currentChannelId={modal.type === 'channelEdit' ? modal.currentChannelId : ''}
      />

      <AddMemberModal
        isOpen={isAddModalOpen}
        onClose={() => { setIsAddModalOpen(false); }}
        onAdd={(data) => {
          // Transform data as needed or pass directly if API expects it
          // API expects Partial<Member>.
          // Data from modal: { name, channelId, nameKo, nameJa }
          // API needs aliases: { ko: [nameKo], ja: [nameJa] } ?
          // Let's adjust data structure
          const memberData: Partial<Member> = {
            name: data.name,
            channelId: data.channelId,
            nameKo: data.nameKo,
            nameJa: data.nameJa,
            aliases: {
              ko: data.nameKo ? [data.nameKo] : [],
              ja: data.nameJa ? [data.nameJa] : []
            },
            isGraduated: false
          }
          addMemberMutation.mutate(memberData)
        }}
      />
    </div >
  )
}

export default MembersTab
