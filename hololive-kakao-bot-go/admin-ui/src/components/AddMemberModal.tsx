import { useState } from 'react'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { X } from 'lucide-react'
import { Button, Input } from '@/components/ui'

interface AddMemberModalProps {
    isOpen: boolean
    onClose: () => void
    onAdd: (member: { name: string; channelId: string; nameKo: string; nameJa: string }) => void
}

const AddMemberModal = ({ isOpen, onClose, onAdd }: AddMemberModalProps) => {
    const [name, setName] = useState('')
    const [channelId, setChannelId] = useState('')
    const [nameKo, setNameKo] = useState('')
    const [nameJa, setNameJa] = useState('')

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        onAdd({ name, channelId, nameKo, nameJa })
        setName('')
        setChannelId('')
        setNameKo('')
        setNameJa('')
        onClose()
    }

    return (
        <Dialog open={isOpen} onClose={onClose} className="relative z-50">
            <div className="fixed inset-0 bg-black/30 backdrop-blur-sm" aria-hidden="true" />
            <div className="fixed inset-0 flex items-center justify-center p-4">
                <DialogPanel className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl border border-slate-100">
                    <div className="flex items-center justify-between mb-6">
                        <DialogTitle className="text-lg font-bold text-slate-900">새 멤버 추가</DialogTitle>
                        <button onClick={onClose} className="p-1 rounded-full hover:bg-slate-100 text-slate-400 hover:text-slate-500 transition-colors">
                            <X size={20} />
                        </button>
                    </div>

                    <form onSubmit={handleSubmit} className="space-y-4">
                        <div>
                            <label className="block text-sm font-bold text-slate-700 mb-1.5">이름 (영어 ID)</label>
                            <Input
                                value={name}
                                onChange={(e) => { setName(e.target.value); }}
                                required
                                placeholder="e.g. suisei (슬러그 역할)"
                                className="w-full bg-slate-50"
                            />
                            <p className="text-xs text-slate-500 mt-1">시스템 내부에서 사용할 고유 식별자입니다.</p>
                        </div>

                        <div>
                            <label className="block text-sm font-bold text-slate-700 mb-1.5">채널 ID</label>
                            <Input
                                value={channelId}
                                onChange={(e) => { setChannelId(e.target.value); }}
                                required
                                placeholder="UC..."
                                className="w-full bg-slate-50 font-mono text-sm"
                            />
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className="block text-sm font-bold text-slate-700 mb-1.5">한국어 이름</label>
                                <Input
                                    value={nameKo}
                                    onChange={(e) => { setNameKo(e.target.value); }}
                                    placeholder="예: 스이세이"
                                    className="w-full bg-slate-50"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-bold text-slate-700 mb-1.5">일본어 이름</label>
                                <Input
                                    value={nameJa}
                                    onChange={(e) => { setNameJa(e.target.value); }}
                                    placeholder="예: すいせい"
                                    className="w-full bg-slate-50"
                                />
                            </div>
                        </div>

                        <div className="flex justify-end gap-3 mt-8 pt-4 border-t border-slate-100">
                            <Button variant="secondary" onClick={onClose} type="button">취소</Button>
                            <Button variant="primary" type="submit">멤버 추가</Button>
                        </div>
                    </form>
                </DialogPanel>
            </div>
        </Dialog>
    )
}

export default AddMemberModal
