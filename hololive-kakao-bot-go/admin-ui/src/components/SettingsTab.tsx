import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { settingsApi, watchdogApi, type WatchdogContainer } from '../api'
import { Save, Settings as SettingsIcon, RefreshCw, Server, Play, Square, AlertCircle, CheckCircle, Power, AlertTriangle } from 'lucide-react'
import { useState, useEffect } from 'react'
import clsx from 'clsx'
import toast, { Toaster } from 'react-hot-toast'

const SettingsTab = () => {
    const queryClient = useQueryClient()
    const { data: settingsData } = useQuery({
        queryKey: ['settings'],
        queryFn: settingsApi.get
    })

    const { data: watchdogHealth } = useQuery({
        queryKey: ['watchdog-health'],
        queryFn: watchdogApi.checkHealth,
        refetchInterval: 30000,
        retry: 1
    })

    const { data: containersData, isLoading: containersLoading, refetch: refetchContainers } = useQuery({
        queryKey: ['watchdog-containers'],
        queryFn: watchdogApi.getContainers,
        enabled: watchdogHealth?.available === true,
        refetchInterval: 15000
    })

    const [alarmAdvanceMinutes, setAlarmAdvanceMinutes] = useState(5)
    const [restartingContainer, setRestartingContainer] = useState<string | null>(null)
    const [confirmModal, setConfirmModal] = useState<{ isOpen: boolean; containerName: string | null }>({
        isOpen: false,
        containerName: null
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
        onError: (err) => {
            toast.error(`설정 저장 실패: ${err.message}`)
        }
    })

    const restartMutation = useMutation({
        mutationFn: (containerName: string) =>
            watchdogApi.restartContainer(containerName, 'Admin dashboard restart'),
        onSuccess: (_, containerName) => {
            setRestartingContainer(null)
            toast.success(
                <span>
                    <span className="font-bold text-slate-800">{containerName}</span>
                    <span className="text-slate-600"> 재시작을 요청했습니다.</span>
                </span>
            )
            void queryClient.invalidateQueries({ queryKey: ['watchdog-containers'] })
        },
        onError: (_, containerName) => {
            setRestartingContainer(null)
            toast.error(
                <span>
                    <span className="font-bold">{containerName}</span> 재시작 실패
                </span>
            )
        }
    })

    const handleSave = () => {
        updateMutation.mutate({
            alarmAdvanceMinutes
        })
    }

    const openConfirmModal = (containerName: string) => {
        setConfirmModal({ isOpen: true, containerName })
    }

    const closeConfirmModal = () => {
        setConfirmModal({ isOpen: false, containerName: null })
    }

    const handleConfirmRestart = () => {
        if (confirmModal.containerName) {
            const name = confirmModal.containerName
            setRestartingContainer(name)
            restartMutation.mutate(name)
            closeConfirmModal()
        }
    }

    const getStateIcon = (state: string, health: string) => {
        if (state !== 'running') return <Square size={14} className="text-red-500" />
        if (health === 'healthy') return <CheckCircle size={14} className="text-emerald-500" />
        if (health === 'unhealthy') return <AlertCircle size={14} className="text-amber-500" />
        return <Play size={14} className="text-sky-500" />
    }

    // 주요 컨테이너만 필터링 (Managed + Watchdog + 핵심 인프라)
    const filteredContainers = containersData?.containers.filter((c: WatchdogContainer) =>
        c.managed ||
        c.name.includes('watchdog') ||
        c.name === 'valkey-mq' ||
        c.name === 'valkey-cache'
    ) ?? []

    return (
        <div className="max-w-3xl mx-auto space-y-6">
            <Toaster
                position="top-center"
                reverseOrder={false}
                toastOptions={{
                    className: 'text-sm font-medium',
                    style: {
                        background: '#ffffff',
                        color: '#334155', // slate-700
                        padding: '12px 16px',
                        borderRadius: '12px',
                        boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
                        border: '1px solid #f1f5f9', // slate-100
                    },
                    success: {
                        iconTheme: {
                            primary: '#0ea5e9', // sky-500
                            secondary: '#ffffff',
                        },
                    },
                    error: {
                        iconTheme: {
                            primary: '#ef4444', // red-500
                            secondary: '#ffffff',
                        },
                    },
                }}
            />

            {/* Confirm Modal (Center) */}
            {confirmModal.isOpen && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm transition-all animate-in fade-in duration-200">
                    <div className="bg-white rounded-xl shadow-2xl w-full max-w-sm overflow-hidden scale-100 animate-in zoom-in-95 duration-200">
                        <div className="p-6">
                            <div className="flex items-center gap-3 mb-4">
                                <div className="bg-amber-100 p-2.5 rounded-full shrink-0">
                                    <AlertTriangle className="text-amber-600" size={24} />
                                </div>
                                <h3 className="text-lg font-bold text-slate-800">컨테이너 재시작</h3>
                            </div>

                            <p className="text-slate-600 mb-2">
                                '<span className="font-bold text-slate-800">{confirmModal.containerName}</span>'을(를) 재시작하시겠습니까?
                            </p>
                            <p className="text-xs text-amber-600 font-medium bg-amber-50 px-3 py-2 rounded-lg">
                                주의: 서비스가 잠시 중단될 수 있습니다.
                            </p>
                        </div>

                        <div className="bg-slate-50 px-6 py-4 flex justify-end gap-3 border-t border-slate-100">
                            <button
                                onClick={closeConfirmModal}
                                className="px-4 py-2 text-sm font-medium text-slate-600 bg-white border border-slate-200 hover:bg-slate-50 rounded-lg transition-colors"
                            >
                                취소
                            </button>
                            <button
                                onClick={handleConfirmRestart}
                                className="px-4 py-2 text-sm font-bold text-white bg-rose-500 hover:bg-rose-600 shadow-sm shadow-rose-200 rounded-lg transition-all active:scale-95"
                            >
                                재시작 실행
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* 알림 설정 */}
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6">
                <div className="flex items-center gap-2 mb-6 border-b border-slate-100 pb-4">
                    <SettingsIcon className="text-slate-600" />
                    <h3 className="text-lg font-bold text-slate-800">시스템 설정</h3>
                </div>

                <div className="space-y-8">
                    <div>
                        <h4 className="text-sm font-bold text-slate-900 mb-4 border-l-2 border-sky-500 pl-3">알림 옵션</h4>

                        <div className="bg-slate-50 rounded-lg p-5 border border-slate-100">
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
                                    className="w-24 px-3 py-2 bg-white border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-sky-500 focus:border-transparent text-slate-700 font-medium text-center"
                                />
                                <span className="text-sm font-medium text-slate-600">분 전 알림</span>
                            </div>
                        </div>
                    </div>

                    <div className="pt-4 flex justify-end border-t border-slate-100">
                        <button
                            onClick={handleSave}
                            disabled={updateMutation.isPending}
                            className={clsx(
                                "bg-sky-500 hover:bg-sky-600 text-white px-5 py-2.5 rounded-lg font-medium flex items-center gap-2 transition-all shadow-sm shadow-sky-200 active:scale-95",
                                "disabled:opacity-70 disabled:cursor-not-allowed"
                            )}
                        >
                            <Save size={18} />
                            {updateMutation.isPending ? '저장 중...' : '변경 사항 저장'}
                        </button>
                    </div>
                </div>
            </div>

            {/* Docker 컨테이너 관리 */}
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6">
                <div className="flex items-center justify-between mb-6 border-b border-slate-100 pb-4">
                    <div className="flex items-center gap-2">
                        <Server className="text-slate-600" />
                        <h3 className="text-lg font-bold text-slate-800">컨테이너 관리</h3>
                        {watchdogHealth?.available ? (
                            <span className="px-2 py-0.5 bg-emerald-100 text-emerald-700 text-xs font-medium rounded-full">
                                Watchdog 연결됨
                            </span>
                        ) : (
                            <span className="px-2 py-0.5 bg-red-100 text-red-700 text-xs font-medium rounded-full">
                                Watchdog 연결 안됨
                            </span>
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
                </div>

                {!watchdogHealth?.available ? (
                    <div className="text-center py-8 text-slate-500 bg-slate-50 rounded-lg border border-slate-100 border-dashed">
                        <AlertCircle size={32} className="mx-auto mb-3 text-slate-400" />
                        <p className="font-medium text-slate-600">Watchdog 서비스에 연결할 수 없습니다</p>
                        <p className="text-xs text-slate-400 mt-1">Docker 관리 기능을 사용하려면 Watchdog 컨테이너가 실행 중이어야 합니다.</p>
                    </div>
                ) : containersLoading ? (
                    <div className="text-center py-12 text-slate-400">
                        <RefreshCw size={24} className="animate-spin mx-auto mb-2 opacity-50" />
                        컨테이너 상태를 불러오는 중...
                    </div>
                ) : filteredContainers.length === 0 ? (
                    <div className="text-center py-8 text-slate-400 bg-slate-50 rounded-lg border border-slate-100 border-dashed">
                        관리 대상 컨테이너가 없습니다.
                    </div>
                ) : (
                    <div className="space-y-3">
                        {filteredContainers.map((container: WatchdogContainer) => (
                            <div
                                key={container.id}
                                className="group flex items-center gap-4 p-4 bg-slate-50 rounded-xl border border-slate-100 hover:bg-white hover:shadow-md hover:border-slate-200 transition-all duration-200"
                            >
                                {/* 1. Icon Section */}
                                <div className="w-12 h-12 rounded-xl bg-white border border-slate-100 flex items-center justify-center shrink-0 shadow-sm group-hover:scale-105 transition-transform duration-200">
                                    {getStateIcon(container.state, container.health)}
                                </div>

                                {/* 2. Information Section */}
                                <div className="flex-1 min-w-0 flex flex-col gap-0.5">
                                    <div className="flex items-center gap-2">
                                        <span className="font-bold text-slate-800 text-sm sm:text-base truncate">
                                            {container.name}
                                        </span>
                                        {container.managed && (
                                            <span className="px-1.5 py-0.5 bg-sky-50 text-sky-600 text-[10px] font-bold rounded-md border border-sky-100 uppercase tracking-tight shrink-0">
                                                Managed
                                            </span>
                                        )}
                                    </div>

                                    <div className="flex items-center gap-2 text-[11px] sm:text-xs">
                                        <div className={clsx(
                                            "px-1.5 py-0.5 rounded font-bold uppercase tracking-wider tabular-nums shrink-0",
                                            container.state === 'running' ? "bg-emerald-50 text-emerald-600" : "bg-rose-50 text-rose-600"
                                        )}>
                                            {container.state}
                                        </div>

                                        {container.health && container.health !== 'none' && (
                                            <div className={clsx(
                                                "px-1.5 py-0.5 rounded font-bold uppercase tracking-wider shrink-0",
                                                container.health === 'healthy' ? "bg-sky-50 text-sky-600" : "bg-amber-50 text-amber-600"
                                            )}>
                                                {container.health}
                                            </div>
                                        )}

                                        <span className="text-slate-300 pointer-events-none shrink-0">•</span>

                                        <span className="truncate text-slate-400 font-medium" title={container.image}>
                                            {container.image.split(':')[0].split('/').pop() ?? 'unknown'}
                                        </span>
                                    </div>
                                </div>

                                {/* 3. Action Section */}
                                <div className="shrink-0">
                                    <button
                                        onClick={() => { openConfirmModal(container.name); }}
                                        disabled={restartingContainer === container.name || restartMutation.isPending}
                                        className={clsx(
                                            "min-w-[84px] h-9 px-3 rounded-lg text-xs font-bold flex items-center justify-center gap-1.5 transition-all border",
                                            restartingContainer === container.name
                                                ? "bg-amber-50 text-amber-600 border-amber-100 cursor-wait"
                                                : "bg-white text-slate-600 border-slate-200 hover:border-rose-200 hover:text-rose-600 hover:bg-rose-50 active:scale-95 shadow-sm"
                                        )}
                                    >
                                        {restartingContainer === container.name ? (
                                            <>
                                                <RefreshCw size={14} className="animate-spin" />
                                                <span>...</span>
                                            </>
                                        ) : (
                                            <>
                                                <Power size={14} />
                                                <span>재시작</span>
                                            </>
                                        )}
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    )
}

export default SettingsTab
