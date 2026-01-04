import { useTwentyQSessions, useTwentyQDeleteSession, useTwentyQInjectHint } from '@/hooks/useGameBots'
import { Skeleton } from '@/components/ui/Skeleton'

import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Badge } from '@/components/ui/Badge'
import { Clock, MessageCircle, Trash2, Zap } from 'lucide-react'
import { ConfirmModal } from '@/components/ConfirmModal'
import { useState } from 'react'
import toast from 'react-hot-toast'

export default function TwentyQSessionsTable() {
    const { data, isLoading } = useTwentyQSessions()
    const deleteSession = useTwentyQDeleteSession()
    const injectHint = useTwentyQInjectHint()

    // Modal State
    const [deleteId, setDeleteId] = useState<string | null>(null)

    if (isLoading) {
        return (
            <div className="space-y-6">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <Skeleton className="w-6 h-6 rounded-full" />
                        <Skeleton className="h-7 w-32" />
                    </div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                    {[...Array(6)].map((_, i) => (
                        <div key={i} className="bg-white rounded-xl border border-slate-200 shadow-sm p-5 space-y-4">
                            <div className="flex justify-between items-start">
                                <div className="space-y-2">
                                    <Skeleton className="h-5 w-16" />
                                    <Skeleton className="h-6 w-32" />
                                </div>
                                <Skeleton className="h-6 w-16" />
                            </div>
                            <Skeleton className="h-3 w-48" />
                            <div className="pt-2 flex items-center gap-2 border-t border-slate-50">
                                <Skeleton className="h-8 flex-1" />
                                <Skeleton className="h-8 flex-1" />
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        )
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

    const handleInjectHint = async (chatId: string) => {
        const hint = window.prompt('주입할 힌트 메시지를 입력하세요:')
        if (!hint) return

        try {
            await injectHint.mutateAsync({ chatId, request: { message: hint } })
            toast.success('힌트가 주입되었습니다.')
        } catch (e) {
            toast.error('힌트 주입 실패')
        }
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-xl font-bold text-slate-800 flex items-center gap-2">
                    <Clock className="w-6 h-6 text-sky-500" />
                    진행 중인 세션
                    <span className="text-sm font-normal text-slate-500 bg-slate-100 px-2 py-0.5 rounded-full ml-2">
                        {sessions.length}
                    </span>
                </h2>
            </div>

            {sessions.length === 0 ? (
                <div className="bg-white p-12 rounded-2xl shadow-sm border border-slate-100 text-center text-slate-400 flex flex-col items-center">
                    <MessageCircle className="w-12 h-12 mb-4 opacity-20" />
                    <p>현재 진행 중인 게임이 없습니다.</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                    {sessions.map((session) => (
                        <Card key={session.chatId} className="hover:shadow-md transition-shadow">
                            <div className="p-5 space-y-4">
                                <div className="flex items-start justify-between">
                                    <div>
                                        <Badge variant="outline" className="mb-2 bg-sky-50 text-sky-600 border-sky-100">
                                            {session.category}
                                        </Badge>
                                        <h3 className="font-bold text-lg text-slate-800 truncate" title={session.target}>
                                            {session.target}
                                        </h3>
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
                                        onClick={() => handleInjectHint(session.chatId)}
                                    >
                                        <Zap className="w-4 h-4 mr-1" />
                                        힌트
                                    </Button>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        className="flex-1 text-rose-500 hover:text-rose-600 hover:bg-rose-50"
                                        onClick={() => setDeleteId(session.chatId)}
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
                message="이 게임 세션을 정말로 강제 종료하시겠습니까? 플레이어들은 더 이상 게임을 진행할 수 없게 됩니다."
                confirmText="종료하기"
                confirmColor="danger"
            />
        </div>
    )
}
