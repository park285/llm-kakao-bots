import { Badge, Button, Card, Input } from '@/components/ui'
import type { Member } from '@/types'
import { Edit2, ExternalLink, GraduationCap, Plus, RotateCcw } from 'lucide-react'

type MemberCardProps = {
  member: Member
  inputs: Record<string, string>
  onInputChange: (key: string, value: string) => void
  onAddAlias: (memberId: number, type: 'ko' | 'ja') => void
  onRemoveAlias: (memberId: number, type: 'ko' | 'ja', alias: string) => void
  onToggleGraduation: (memberId: number, memberName: string, currentStatus: boolean) => void
  onEditChannel: (memberId: number, memberName: string, currentChannelId: string) => void
  onEditName: (memberId: number, currentName: string) => void
}

const MemberCard = ({
  member,
  inputs,
  onInputChange,
  onAddAlias,
  onRemoveAlias,
  onToggleGraduation,
  onEditChannel,
  onEditName,
}: MemberCardProps) => (
  <Card className="relative group overflow-hidden border-slate-200">
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
          <div className="flex items-center gap-1.5">
            <h3 className="font-bold text-lg text-slate-800 leading-tight">{member.name}</h3>
            <button
              onClick={() => { onEditName(member.id, member.name) }}
              className="opacity-0 group-hover:opacity-100 p-1 text-slate-400 hover:text-sky-600 transition-all"
              title="이름 수정"
            >
              <Edit2 size={14} />
            </button>
          </div>
        </div>

        <button
          onClick={() => { onToggleGraduation(member.id, member.name, member.isGraduated) }}
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
            onEditChannel(member.id, member.name, member.channelId)
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
              onRemove={() => { onRemoveAlias(member.id, 'ko', alias) }}
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
            onChange={(e) => { onInputChange(`${String(member.id)}-ko`, e.target.value) }}
            placeholder="별명 추가"
            className="flex-1 h-8 text-xs bg-slate-50 border-slate-200"
            onKeyDown={(e) => { if (e.key === 'Enter') onAddAlias(member.id, 'ko') }}
          />
          <Button
            variant="primary"
            size="sm"
            onClick={() => { onAddAlias(member.id, 'ko') }}
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
              onRemove={() => { onRemoveAlias(member.id, 'ja', alias) }}
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
            onChange={(e) => { onInputChange(`${String(member.id)}-ja`, e.target.value) }}
            placeholder="エイリアス追加"
            className="flex-1 h-8 text-xs bg-slate-50 border-slate-200"
            onKeyDown={(e) => { if (e.key === 'Enter') onAddAlias(member.id, 'ja') }}
          />
          <Button
            variant="primary"
            size="sm"
            onClick={() => { onAddAlias(member.id, 'ja') }}
            className="h-8 w-8 p-0 flex items-center justify-center bg-rose-500 hover:bg-rose-600"
          >
            <Plus size={14} />
          </Button>
        </div>
      </div>
    </Card.Body>
  </Card>
)

export default MemberCard
