import { useState, useEffect, useCallback } from 'react';
import { getWatchdogStatus, reloadConfig, triggerCheckNow, setWatchdogEnabled, WatchdogStatus } from '@/api/client';
import { Loader2, RotateCcw, Monitor, Zap, CheckCircle, XCircle, Clock, Server, Activity, FileJson, Power } from 'lucide-react';
import { motion } from 'framer-motion';
import { useToast } from '@/stores/toastStore';
import { cn } from '@/utils';

export function SettingsPage() {
    const [status, setStatus] = useState<WatchdogStatus | null>(null);
    const [loading, setLoading] = useState(true);
    const [reloading, setReloading] = useState(false);
    const [checking, setChecking] = useState(false);
    const { addToast } = useToast();

    const loadStatus = useCallback(async () => {
        try {
            const data = await getWatchdogStatus();
            setStatus(data);
        } catch (err) {
            addToast('error', `설정 로드 실패: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setLoading(false);
        }
    }, [addToast]);

    useEffect(function load() {
        void loadStatus();
        const interval = setInterval(() => { void loadStatus(); }, 10000);
        return function cleanup() { clearInterval(interval); };
    }, [loadStatus]);

    async function handleReload() {
        if (!confirm('설정 파일을 다시 로드하시겠습니까? 현재 런타임 설정이 파일 내용으로 교체됩니다.')) return;
        setReloading(true);
        try {
            const result = await reloadConfig();
            addToast('success', `설정 리로드 완료 (적용된 필드: ${result.appliedFields?.length || 0}개)`);
            await loadStatus();
        } catch (err) {
            addToast('error', `설정 리로드 실패: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setReloading(false);
        }
    }

    async function handleCheckNow() {
        setChecking(true);
        try {
            await triggerCheckNow();
            addToast('success', '헬스체크가 트리거되었습니다.');
        } catch (err) {
            addToast('error', `헬스체크 트리거 실패: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setChecking(false);
        }
    }

    async function handleToggleEnabled() {
        if (!status) return;
        const newEnabled = !status.enabled;
        const action = newEnabled ? '활성화' : '비활성화';
        if (!confirm(`정말로 Watchdog을 전체 ${action} 하시겠습니까?`)) return;

        try {
            await setWatchdogEnabled(newEnabled, `admin_ui_toggle`);
            addToast('success', `Watchdog이 ${action} 되었습니다.`);
            await loadStatus();
        } catch (err) {
            addToast('error', `변경 실패: ${err instanceof Error ? err.message : 'Unknown'}`);
        }
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64 text-slate-400">
                <Loader2 className="animate-spin mr-2" />
                Loading configuration...
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold text-slate-800 flex items-center gap-2">
                <Monitor className="text-slate-500" />
                Watchdog Settings
            </h1>

            {/* Quick Actions */}
            <motion.div
                animate={{ opacity: 1, y: 0 }}
                className="grid grid-cols-1 md:grid-cols-3 gap-4"
            >
                <button
                    onClick={handleToggleEnabled}
                    className={cn(
                        "p-4 rounded-2xl text-white text-left hover:shadow-lg transition-all group relative overflow-hidden",
                        status?.enabled
                            ? "bg-gradient-to-br from-emerald-500 to-teal-600 hover:shadow-emerald-200"
                            : "bg-gradient-to-br from-rose-500 to-red-600 hover:shadow-rose-200"
                    )}
                >
                    <div className="absolute top-0 right-0 p-4 opacity-10">
                        <Power size={64} />
                    </div>
                    <div className="flex items-center gap-3 relative z-10">
                        <div className={cn(
                            "p-2 rounded-full",
                            status?.enabled ? "bg-emerald-400/30" : "bg-rose-400/30"
                        )}>
                            <Power size={24} className="text-white" />
                        </div>
                        <div>
                            <h3 className="font-bold">{status?.enabled ? 'System Online' : 'System Offline'}</h3>
                            <p className="text-sm text-white/80">
                                {status?.enabled ? 'Click to disable watchdog' : 'Click to enable monitoring'}
                            </p>
                        </div>
                    </div>
                </button>

                <button
                    onClick={handleCheckNow}
                    disabled={checking}
                    className="p-4 bg-gradient-to-br from-sky-500 to-indigo-600 rounded-2xl text-white text-left hover:shadow-lg hover:shadow-indigo-200 transition-all group"
                >
                    <div className="flex items-center gap-3">
                        {checking ? (
                            <Loader2 size={24} className="animate-spin" />
                        ) : (
                            <Zap size={24} className="group-hover:scale-110 transition-transform" />
                        )}
                        <div>
                            <h3 className="font-bold">Trigger Health Check</h3>
                            <p className="text-sm text-white/80">즉시 모든 관리대상 컨테이너 헬스체크 실행</p>
                        </div>
                    </div>
                </button>

                <button
                    onClick={handleReload}
                    disabled={reloading}
                    className="p-4 bg-gradient-to-br from-amber-400 to-orange-500 rounded-2xl text-white text-left hover:shadow-lg hover:shadow-orange-200 transition-all group"
                >
                    <div className="flex items-center gap-3">
                        {reloading ? (
                            <Loader2 size={24} className="animate-spin" />
                        ) : (
                            <RotateCcw size={24} className="group-hover:rotate-180 transition-transform duration-500" />
                        )}
                        <div>
                            <h3 className="font-bold">Reload Config</h3>
                            <p className="text-sm text-white/80">설정 파일(config.json)에서 설정 다시 로드</p>
                        </div>
                    </div>
                </button>
            </motion.div>

            {/* Status Overview */}
            {status && (
                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.1 }}
                    className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden"
                >
                    <div className="px-6 py-4 border-b border-slate-100 bg-slate-50/50 flex justify-between items-center">
                        <div>
                            <h3 className="font-semibold text-slate-800">Current Status</h3>
                            <p className="text-xs text-slate-500">Runtime configuration overview</p>
                        </div>
                        <span className={cn(
                            "px-3 py-1 rounded-full text-xs font-bold",
                            status.enabled
                                ? "bg-emerald-50 text-emerald-600 border border-emerald-200"
                                : "bg-rose-50 text-rose-600 border border-rose-200"
                        )}>
                            {status.enabled ? (
                                <><CheckCircle size={12} className="inline mr-1" />ENABLED</>
                            ) : (
                                <><XCircle size={12} className="inline mr-1" />DISABLED</>
                            )}
                        </span>
                    </div>

                    <div className="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        <ConfigItem icon={<Clock size={16} />} label="Uptime" value={status.uptime} />
                        <ConfigItem icon={<Server size={16} />} label="Managed Containers" value={String(status.containers?.length || 0)} />
                        <ConfigItem icon={<Activity size={16} />} label="Check Interval" value={`${status.intervalSec}s`} />
                        <ConfigItem icon={<FileJson size={16} />} label="Config Source" value={status.configSource} />
                        <ConfigItem icon={<Activity size={16} />} label="Max Failures" value={String(status.maxFailures)} />
                        <ConfigItem icon={<Clock size={16} />} label="Cooldown" value={`${status.cooldownSec}s`} />
                        <ConfigItem icon={<Clock size={16} />} label="Restart Timeout" value={`${status.restartTimeoutSec}s`} />
                        <ConfigItem icon={<Activity size={16} />} label="Use Events" value={status.useEvents ? 'Yes' : 'No'} />
                        <ConfigItem icon={<Activity size={16} />} label="Verbose Logging" value={status.verbose ? 'Yes' : 'No'} />
                    </div>
                </motion.div>
            )}

            {/* Managed Containers List */}
            {status && status.containers && status.containers.length > 0 && (
                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.2 }}
                    className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden"
                >
                    <div className="px-6 py-4 border-b border-slate-100 bg-slate-50/50">
                        <h3 className="font-semibold text-slate-800">Managed Containers</h3>
                        <p className="text-xs text-slate-500">설정 파일에 정의된 관리 대상 컨테이너들</p>
                    </div>
                    <div className="p-6">
                        <div className="flex flex-wrap gap-2">
                            {status.containers.map((name) => (
                                <span
                                    key={name}
                                    className="px-3 py-1.5 bg-indigo-50 text-indigo-700 rounded-lg text-sm font-medium border border-indigo-100"
                                >
                                    {name}
                                </span>
                            ))}
                        </div>
                    </div>
                </motion.div>
            )}

            {/* Raw Config JSON */}
            {status && (
                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.3 }}
                    className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden"
                >
                    <div className="px-6 py-4 border-b border-slate-100 bg-slate-50/50">
                        <h3 className="font-semibold text-slate-800">Raw Configuration</h3>
                        <p className="text-xs text-slate-500">
                            {status.configPath ? `File: ${status.configPath}` : 'Environment variables only'}
                        </p>
                    </div>
                    <div className="p-6">
                        <pre className="font-mono text-xs bg-slate-900 text-slate-200 p-4 rounded-xl overflow-x-auto leading-relaxed">
                            {JSON.stringify(status, null, 2)}
                        </pre>
                    </div>
                </motion.div>
            )}
        </div>
    );
}

function ConfigItem({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
    return (
        <div className="flex items-center gap-3 p-3 bg-slate-50 rounded-xl">
            <div className="text-slate-400">{icon}</div>
            <div>
                <p className="text-xs text-slate-500 uppercase tracking-wider">{label}</p>
                <p className="font-medium text-slate-800">{value}</p>
            </div>
        </div>
    );
}
