import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { settingsApi, dockerApi, type DockerContainer } from '@/api'
import { Save, Settings as SettingsIcon, RefreshCw, Server, Play, Square, AlertCircle, Power, AlertTriangle, StopCircle } from 'lucide-react'
import { useState, useEffect } from 'react'
import clsx from 'clsx'
import toast, { Toaster } from 'react-hot-toast'
import { Card, Badge, Button } from '@/components/ui'

const SettingsTab = () => {
    const queryClient = useQueryClient()
    const { data: settingsData } = useQuery({
        queryKey: ['settings'],
        queryFn: settingsApi.get
    })

    const { data: dockerHealth } = useQuery({
        queryKey: ['docker-health'],
        queryFn: dockerApi.checkHealth,
        refetchInterval: 30000,
        retry: 1
    })

    const { data: containersData, isLoading: containersLoading, refetch: refetchContainers } = useQuery({
        queryKey: ['docker-containers'],
        queryFn: dockerApi.getContainers,
        enabled: dockerHealth?.available === true,
        refetchInterval: 15000
    })

    const [alarmAdvanceMinutes, setAlarmAdvanceMinutes] = useState(5)
    const [actionInProgress, setActionInProgress] = useState<string | null>(null)
    const [confirmModal, setConfirmModal] = useState<{ isOpen: boolean; containerName: string | null; action: 'restart' | 'stop' | 'start' | null }>({
        isOpen: false,
        containerName: null,
        action: null
    })

    useEffect(() => {
        if (settingsData?.settings) {
            setAlarmAdvanceMinutes(settingsData.settings.alarmAdvanceMinutes)
        }
    }, [settingsData])

    const updateMutation = useMutation({
        mutationFn: settingsApi.update,
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: ['settings'] })
            toast.success('설정이 성공적으로 저장되었습니다.')
        },
        onError: (err: Error) => {
            toast.error(`설정 저장 실패: ${err.message}`)
        }
    })

    const restartMutation = useMutation({
        mutationFn: (containerName: string) => dockerApi.restartContainer(containerName),
        onSuccess: (_: unknown, containerName: string) => {
            setActionInProgress(null)
            toast.success(
                <span>
                    <span className="font-bold text-slate-800">{containerName}</span>
                    <span className="text-slate-600"> 재시작을 요청했습니다.</span>
                </span>
            )
            void queryClient.invalidateQueries({ queryKey: ['docker-containers'] })
        },
        onError: (_: unknown, containerName: string) => {
            setActionInProgress(null)
            toast.error(
                <span>
                    <span className="font-bold">{containerName}</span> 재시작 실패
                </span>
            )
        }
    })

    const stopMutation = useMutation({
        mutationFn: (containerName: string) => dockerApi.stopContainer(containerName),
        onSuccess: (_: unknown, containerName: string) => {
            setActionInProgress(null)
            toast.success(
                <span>
                    <span className="font-bold text-slate-800">{containerName}</span>
                    <span className="text-slate-600"> 중지되었습니다.</span>
                </span>
            )
            void queryClient.invalidateQueries({ queryKey: ['docker-containers'] })
        },
        onError: (_: unknown, containerName: string) => {
            setActionInProgress(null)
            toast.error(
                <span>
                    <span className="font-bold">{containerName}</span> 중지 실패
                </span>
            )
        }
    })

    const startMutation = useMutation({
        mutationFn: (containerName: string) => dockerApi.startContainer(containerName),
        onSuccess: (_: unknown, containerName: string) => {
            setActionInProgress(null)
            toast.success(
                <span>
                    <span className="font-bold text-slate-800">{containerName}</span>
                    <span className="text-slate-600"> 시작되었습니다.</span>
                </span>
            )
            void queryClient.invalidateQueries({ queryKey: ['docker-containers'] })
        },
        onError: (_: unknown, containerName: string) => {
            setActionInProgress(null)
            toast.error(
                <span>
                    <span className="font-bold">{containerName}</span> 시작 실패
                </span>
            )
        }
    })

    const handleSave = () => {
        updateMutation.mutate({
            alarmAdvanceMinutes
        })
    }

    const openConfirmModal = (containerName: string, action: 'restart' | 'stop' | 'start') => {
        setConfirmModal({ isOpen: true, containerName, action })
    }

    const closeConfirmModal = () => {
        setConfirmModal({ isOpen: false, containerName: null, action: null })
    }

    const handleConfirmAction = () => {
        if (confirmModal.containerName && confirmModal.action) {
            const name = confirmModal.containerName
            setActionInProgress(name)

            switch (confirmModal.action) {
                case 'restart':
                    restartMutation.mutate(name)
                    break
                case 'stop':
                    stopMutation.mutate(name)
                    break
                case 'start':
                    startMutation.mutate(name)
                    break
            }
            closeConfirmModal()
        }
    }

    const getActionLabel = (action: 'restart' | 'stop' | 'start' | null) => {
        switch (action) {
            case 'restart': return '재시작'
            case 'stop': return '중지'
            case 'start': return '시작'
            default: return ''
        }
    }

    const containers = containersData?.containers ?? []

    return (
        <div className="max-w-4xl mx-auto space-y-6">
            <Toaster
                position="top-center"
                reverseOrder={false}
                toastOptions={{
                    className: 'text-sm font-medium',
                    style: {
                        background: '#ffffff',
                        color: '#334155',
                        padding: '12px 16px',
                        borderRadius: '12px',
                        boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
                        border: '1px solid #f1f5f9',
                    },
                    success: {
                        iconTheme: {
                            primary: '#0ea5e9',
                            secondary: '#ffffff',
                        },
                    },
                    error: {
                        iconTheme: {
                            primary: '#ef4444',
                            secondary: '#ffffff',
                        },
                    },
                }}
            />

            {/* Confirm Modal */}
            {confirmModal.isOpen && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm transition-all animate-in fade-in duration-200">
                    <div className="bg-white rounded-xl shadow-2xl w-full max-w-sm overflow-hidden scale-100 animate-in zoom-in-95 duration-200 border border-slate-200">
                        <div className="p-6">
                            <div className="flex items-center gap-3 mb-4">
                                <div className="bg-amber-100 p-2.5 rounded-full shrink-0">
                                    <AlertTriangle className="text-amber-600" size={24} />
                                </div>
                                <h3 className="text-lg font-bold text-slate-800">컨테이너 {getActionLabel(confirmModal.action)}</h3>
                            </div>

                            <p className="text-slate-600 mb-2">
                                '<span className="font-bold text-slate-800">{confirmModal.containerName}</span>'을(를) {getActionLabel(confirmModal.action)}하시겠습니까?
                            </p>
                            {confirmModal.action !== 'start' && (
                                <p className="text-xs text-amber-600 font-medium bg-amber-50 px-3 py-2 rounded-lg">
                                    주의: 서비스가 잠시 중단될 수 있습니다.
                                </p>
                            )}
                        </div>

                        <div className="bg-slate-50 px-6 py-4 flex justify-end gap-3 border-t border-slate-100">
                            <Button
                                variant="secondary"
                                onClick={closeConfirmModal}
                            >
                                취소
                            </Button>
                            <Button
                                className={clsx(
                                    "text-white",
                                    confirmModal.action === 'stop' ? "bg-rose-600 hover:bg-rose-700" :
                                        confirmModal.action === 'start' ? "bg-emerald-600 hover:bg-emerald-700" :
                                            "bg-amber-500 hover:bg-amber-600"
                                )}
                                onClick={handleConfirmAction}
                            >
                                {getActionLabel(confirmModal.action)} 실행
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {/* 알림 설정 */}
            <Card>
                <Card.Header className="flex items-center gap-2 border-b border-slate-100 pb-4">
                    <SettingsIcon className="text-slate-600" size={20} />
                    <h3 className="text-lg font-bold text-slate-800">시스템 설정</h3>
                </Card.Header>

                <Card.Body className="space-y-6 pt-6">
                    <div>
                        <h4 className="text-sm font-bold text-slate-900 mb-4 border-l-2 border-sky-500 pl-3">알림 옵션</h4>

                        <div className="bg-slate-50 rounded-lg p-5 border border-slate-100 hover:border-slate-200 transition-colors">
                            <label className="block text-sm font-medium text-slate-700 mb-2">
                                알람 사전 알림 시간
                            </label>
                            <p className="text-xs text-slate-500 mb-3">
                                방송 시작 몇 분 전에 채팅방으로 알람을 전송할지 설정합니다.
                            </p>
                            <div className="flex items-center gap-3">
                                <input
                                    type="number"
                                    min="1"
                                    max="60"
                                    value={alarmAdvanceMinutes}
                                    onChange={(e) => { setAlarmAdvanceMinutes(parseInt(e.target.value) || 0); }}
                                    className="w-24 px-3 py-2 bg-white border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-sky-500 focus:border-transparent text-slate-700 font-medium text-center shadow-sm"
                                />
                                <span className="text-sm font-medium text-slate-600">분 전 알림</span>
                            </div>
                        </div>
                    </div>

                    <div className="flex justify-end pt-2">
                        <Button
                            onClick={handleSave}
                            disabled={updateMutation.isPending}
                            className="gap-2"
                        >
                            <Save size={16} />
                            {updateMutation.isPending ? '저장 중...' : '변경 사항 저장'}
                        </Button>
                    </div>
                </Card.Body>
            </Card>

            {/* Docker 컨테이너 관리 */}
            <Card>
                <Card.Header className="flex items-center justify-between border-b border-slate-100 pb-4">
                    <div className="flex items-center gap-2">
                        <Server className="text-slate-600" size={20} />
                        <h3 className="text-lg font-bold text-slate-800">컨테이너 관리</h3>
                        {dockerHealth?.available ? (
                            <Badge color="green" className="px-2 py-0.5">Docker 연결됨</Badge>
                        ) : (
                            <Badge color="rose" className="px-2 py-0.5">Docker 연결 안됨</Badge>
                        )}
                    </div>
                    <button
                        onClick={() => { void refetchContainers(); }}
                        disabled={containersLoading}
                        className="p-2 hover:bg-slate-100 rounded-lg text-slate-500 transition-colors"
                        title="목록 새로고침"
                    >
                        <RefreshCw size={18} className={containersLoading ? 'animate-spin' : ''} />
                    </button>
                </Card.Header>

                <Card.Body className="pt-6">
                    {!dockerHealth?.available ? (
                        <div className="text-center py-10 text-slate-500 bg-slate-50 rounded-xl border border-slate-100 border-dashed">
                            <AlertCircle size={32} className="mx-auto mb-3 text-slate-400" />
                            <p className="font-medium text-slate-600">Docker 서비스에 연결할 수 없습니다</p>
                            <p className="text-xs text-slate-400 mt-1">Docker 소켓이 마운트되어 있는지 확인하세요.</p>
                        </div>
                    ) : containersLoading ? (
                        <div className="text-center py-12 text-slate-400">
                            <RefreshCw size={24} className="animate-spin mx-auto mb-2 opacity-50" />
                            컨테이너 상태를 불러오는 중...
                        </div>
                    ) : containers.length === 0 ? (
                        <div className="text-center py-10 text-slate-400 bg-slate-50 rounded-xl border border-slate-100 border-dashed">
                            관리 대상 컨테이너가 없습니다.
                        </div>
                    ) : (
                        <div className="grid gap-3">
                            {containers.map((container: DockerContainer) => (
                                <div
                                    key={container.id}
                                    className="group flex flex-col sm:flex-row items-start sm:items-center gap-4 p-4 bg-slate-50 rounded-xl border border-slate-100 hover:bg-white hover:shadow-md hover:border-slate-200 transition-all duration-200"
                                >
                                    {/* Icon Section */}
                                    <div className={clsx(
                                        "w-12 h-12 rounded-xl border flex items-center justify-center shrink-0 shadow-sm transition-colors",
                                        container.state === 'running'
                                            ? "bg-white border-slate-100 text-sky-500"
                                            : "bg-slate-100 border-slate-200 text-slate-400"
                                    )}>
                                        {container.state === 'running'
                                            ? (container.health === 'unhealthy' ? <AlertCircle size={20} className="text-amber-500" /> : <Play size={20} className="fill-current" />)
                                            : <Square size={20} className="fill-current" />
                                        }
                                    </div>

                                    {/* Information Section */}
                                    <div className="flex-1 min-w-0 flex flex-col gap-1 w-full">
                                        <div className="flex items-center justify-between sm:justify-start gap-2">
                                            <span className="font-bold text-slate-800 text-base truncate">
                                                {container.name}
                                            </span>
                                            {container.managed && (
                                                <Badge color="sky">관리됨</Badge>
                                            )}
                                        </div>

                                        <div className="flex items-center gap-2 text-xs flex-wrap">
                                            <Badge
                                                color={container.state === 'running' ? 'green' : 'gray'}
                                                className="uppercase tracking-wider font-bold"
                                            >
                                                {container.state}
                                            </Badge>

                                            {container.health && container.health !== 'none' && (
                                                <Badge
                                                    color={container.health === 'healthy' ? 'sky' : 'amber'}
                                                    className="uppercase tracking-wider font-bold"
                                                >
                                                    {container.health}
                                                </Badge>
                                            )}

                                            <span className="hidden sm:inline text-slate-300 pointer-events-none shrink-0">•</span>

                                            <span className="font-mono text-slate-400 bg-slate-100 px-1.5 py-0.5 rounded text-[10px] truncate max-w-[200px]" title={container.image}>
                                                {container.image.split(':')[0]?.split('/').pop() ?? 'unknown'}
                                            </span>
                                        </div>
                                    </div>

                                    {/* Action Section */}
                                    <div className="shrink-0 flex gap-2 w-full sm:w-auto mt-2 sm:mt-0 justify-end">
                                        {container.state === 'running' ? (
                                            <>
                                                <Button
                                                    size="sm"
                                                    variant="secondary"
                                                    onClick={() => { openConfirmModal(container.name, 'restart'); }}
                                                    disabled={actionInProgress === container.name}
                                                    className={clsx(
                                                        "h-9 px-3 gap-1.5 font-bold hover:bg-amber-50 hover:text-amber-600 hover:border-amber-200",
                                                        actionInProgress === container.name && "cursor-wait opacity-70"
                                                    )}
                                                    title="재시작"
                                                >
                                                    {actionInProgress === container.name ? (
                                                        <RefreshCw size={14} className="animate-spin" />
                                                    ) : (
                                                        <Power size={14} />
                                                    )}
                                                    <span className="sm:hidden lg:inline">재시작</span>
                                                </Button>
                                                <Button
                                                    size="sm"
                                                    variant="secondary"
                                                    onClick={() => { openConfirmModal(container.name, 'stop'); }}
                                                    disabled={actionInProgress === container.name}
                                                    className="h-9 px-3 gap-1.5 font-bold hover:bg-rose-50 hover:text-rose-600 hover:border-rose-200"
                                                    title="중지"
                                                >
                                                    <StopCircle size={14} />
                                                    <span className="sm:hidden lg:inline">중지</span>
                                                </Button>
                                            </>
                                        ) : (
                                            <Button
                                                size="sm"
                                                variant="secondary"
                                                onClick={() => { openConfirmModal(container.name, 'start'); }}
                                                disabled={actionInProgress === container.name}
                                                className="h-9 px-3 gap-1.5 font-bold hover:bg-emerald-50 hover:text-emerald-600 hover:border-emerald-200"
                                                title="시작"
                                            >
                                                <Play size={14} />
                                                <span className="sm:hidden lg:inline">시작</span>
                                            </Button>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </Card.Body>
            </Card>
        </div>
    )
}

export default SettingsTab
