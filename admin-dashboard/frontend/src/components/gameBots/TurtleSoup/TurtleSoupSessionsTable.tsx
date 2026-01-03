import { useTurtleSoupSessions, useTurtleSoupDeleteSession, useTurtleSoupInjectHint } from '@/hooks/useGameBots'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Badge } from '@/components/ui/Badge'
import { Clock, MessageCircle, Trash2, Zap, HelpCircle } from 'lucide-react'
import { ConfirmModal } from '@/components/ConfirmModal'
import { useState } from 'react'
import toast from 'react-hot-toast'

export default function TurtleSoupSessionsTable() {
    const { data, isLoading } = useTurtleSoupSessions()
    const deleteSession = useTurtleSoupDeleteSession()
    const injectHint = useTurtleSoupInjectHint()

    const [deleteId, setDeleteId] = useState<string | null>(null)

    if (isLoading) {
        return <div className="p-8 text-center text-slate-500">세션 목록을 불러오는 중...</div>
    }

    const sessions = data?.sessions || []

    const handleDelete = async () => {
        if (!deleteId) return
        try {
            await deleteSession.mutateAsync(deleteId)
            toast.success('세션이 종료되었습니다.')
        } catch (e) {
            // Error handled by global handler
        } finally {
            setDeleteId(null)
        }
    }

    const handleInjectHint = async (sessionId: string) => {
        const hint = window.prompt('주입할 힌트 메시지를 입력하세요:')
        if (!hint) return

        try {
            await injectHint.mutateAsync({ sessionId, request: { message: hint } })
            toast.success('힌트가 주입되었습니다.')
        } catch (e) {
            toast.error('힌트 주입 실패')
        }
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-xl font-bold text-slate-800 flex items-center gap-2">
                    <Clock className="w-6 h-6 text-emerald-500" />
                    진행 중인 세션
                    <span className="text-sm font-normal text-slate-500 bg-slate-100 px-2 py-0.5 rounded-full ml-2">
                        {sessions.length}
                    </span>
                </h2>
            </div>

            {sessions.length === 0 ? (
                <div className="bg-white p-12 rounded-2xl shadow-sm border border-slate-100 text-center text-slate-400 flex flex-col items-center">
                    <MessageCircle className="w-12 h-12 mb-4 opacity-20" />
                    <p>현재 진행 중인 퍼즐이 없습니다.</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                    {sessions.map((session) => (
                        <Card key={session.sessionId} className="hover:shadow-md transition-shadow">
                            <div className="p-5 space-y-4">
                                <div className="flex items-start justify-between">
                                    <div>
                                        <Badge variant="outline" className="mb-2 bg-emerald-50 text-emerald-600 border-emerald-100">
                                            진행 중
                                        </Badge>
                                        <div className="flex items-center gap-2 text-sm text-slate-600 mt-1">
                                            <HelpCircle className="w-3.5 h-3.5" />
                                            <span>질문: {session.questionCount}</span>
                                            <span className="text-slate-300">|</span>
                                            <span>힌트: {session.hintCount}</span>
                                        </div>
                                    </div>
                                    <div className={`px-2 py-1 rounded text-xs font-bold ${session.ttlSeconds < 60 ? 'bg-rose-100 text-rose-600' : 'bg-slate-100 text-slate-600'
                                        }`}>
                                        {Math.floor(session.ttlSeconds / 60)}분 남음
                                    </div>
                                </div>

                                <div className="text-xs text-slate-400 font-mono break-all line-clamp-1">
                                    ID: {session.chatId}
                                </div>

                                <div className="pt-2 flex items-center gap-2 border-t border-slate-50">
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        className="flex-1 text-amber-600 hover:text-amber-700 hover:bg-amber-50"
                                        onClick={() => handleInjectHint(session.sessionId)}
                                    >
                                        <Zap className="w-4 h-4 mr-1" />
                                        힌트
                                    </Button>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        className="flex-1 text-rose-500 hover:text-rose-600 hover:bg-rose-50"
                                        onClick={() => setDeleteId(session.sessionId)}
                                    >
                                        <Trash2 className="w-4 h-4 mr-1" />
                                        종료
                                    </Button>
                                </div>
                            </div>
                        </Card>
                    ))}
                </div>
            )}

            <ConfirmModal
                isOpen={!!deleteId}
                onClose={() => setDeleteId(null)}
                onConfirm={handleDelete}
                title="세션 강제 종료"
                message="이 퍼즐 세션을 정말로 강제 종료하시겠습니까?"
                confirmText="종료하기"
                confirmColor="danger"
            />
        </div>
    )
}
