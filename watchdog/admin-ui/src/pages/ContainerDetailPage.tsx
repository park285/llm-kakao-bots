import { useParams, Link } from 'react-router-dom';
import { useEffect, useState, useCallback } from 'react';
import { getTargetDetails, restartContainer, startContainer, stopContainer, pauseMonitoring, resumeMonitoring } from '@/api/client';
import { ContainerInfo } from '@/types';
import { LogViewer } from '@/components/LogViewer';
import { cn, getStatusColor } from '@/utils';
import {
    ArrowLeft,
    Play,
    Square,
    RefreshCw,
    Box,
    Loader2,
    AlertCircle,
    Clock,
    Pause,
    CirclePlay,
    ShieldCheck,
    ShieldOff,
    Activity
} from 'lucide-react';
import { motion } from 'framer-motion';
import { useToast } from '@/stores/toastStore';

export function ContainerDetailPage() {
    const { name } = useParams<{ name: string }>();
    const [container, setContainer] = useState<ContainerInfo | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [actionLoading, setActionLoading] = useState<string | null>(null);
    const { addToast } = useToast();

    const fetchDetails = useCallback(async function fetchContainerDetails() {
        if (!name) return;
        try {
            const data = await getTargetDetails(name);
            setContainer(data);
            setError(null);
        } catch (err) {
            if (err instanceof Error) {
                if (err.message.includes('404')) {
                    setError('This container is not a managed target.');
                } else {
                    setError(err.message);
                }
            } else {
                setError('Failed to load container');
            }
        } finally {
            setLoading(false);
        }
    }, [name]);

    useEffect(function loadAndPoll() {
        fetchDetails();
        const interval = setInterval(fetchDetails, 3000);
        return function cleanup() { clearInterval(interval); };
    }, [fetchDetails]);

    async function handleRestart() {
        if (!name) return;
        setActionLoading('restart');
        try {
            await restartContainer(name, false);
            addToast('success', `${name} 재시작 요청됨`);
            await fetchDetails();
        } catch (err) {
            addToast('error', `Restart failed: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setActionLoading(null);
        }
    }

    async function handleStart() {
        if (!name) return;
        setActionLoading('start');
        try {
            await startContainer(name);
            addToast('success', `${name} 시작됨`);
            await fetchDetails();
        } catch (err) {
            addToast('error', `Start failed: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setActionLoading(null);
        }
    }

    async function handleStop() {
        if (!name || !confirm('Are you sure you want to stop this container? Monitoring will be paused.')) return;
        setActionLoading('stop');
        try {
            await stopContainer(name);
            addToast('success', `${name} 중지됨`);
            await fetchDetails();
        } catch (err) {
            addToast('error', `Stop failed: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setActionLoading(null);
        }
    }

    async function handlePauseMonitoring() {
        if (!name) return;
        setActionLoading('pause');
        try {
            await pauseMonitoring(name);
            addToast('info', `${name} 모니터링 일시중지됨`);
            await fetchDetails();
        } catch (err) {
            addToast('error', `Pause failed: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setActionLoading(null);
        }
    }

    async function handleResumeMonitoring() {
        if (!name) return;
        setActionLoading('resume');
        try {
            await resumeMonitoring(name);
            addToast('success', `${name} 모니터링 재개됨`);
            await fetchDetails();
        } catch (err) {
            addToast('error', `Resume failed: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setActionLoading(null);
        }
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64 text-slate-400">
                <Loader2 className="animate-spin mr-2" />
                Loading container details...
            </div>
        );
    }

    if (error || !container) {
        return (
            <div className="p-6 bg-rose-50 border border-rose-200 rounded-2xl text-rose-600 max-w-xl mx-auto">
                <AlertCircle className="mx-auto w-10 h-10 mb-2 opacity-50" />
                <h3 className="font-bold text-center">Cannot Load Container</h3>
                <p className="text-sm text-center mt-1">{error || 'Container not found'}</p>
                <Link to="/containers" className="block text-center mt-4 text-sm text-sky-600 hover:underline">
                    ← Back to Containers
                </Link>
            </div>
        );
    }

    const status = container.status || 'unknown';
    const statusClasses = getStatusColor(status);
    const isPaused = container.monitoringPaused ?? false;

    return (
        <div className="space-y-6">
            {/* Header */}
            <motion.div
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                className="flex flex-col md:flex-row md:items-center justify-between gap-4"
            >
                <div className="flex items-center gap-4">
                    <Link
                        to="/containers"
                        className="p-2 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 transition-colors"
                    >
                        <ArrowLeft size={20} className="text-slate-600" />
                    </Link>
                    <div className="flex items-center gap-3">
                        <div className="p-3 bg-indigo-50 rounded-xl text-indigo-600">
                            <Box size={24} />
                        </div>
                        <div>
                            <h1 className="text-2xl font-bold text-slate-800">{container.name}</h1>
                            <p className="text-xs text-slate-400 font-mono">{container.id}</p>
                        </div>
                    </div>
                </div>

                {/* Status badges */}
                <div className="flex items-center gap-2">
                    <div className={cn("px-3 py-1.5 rounded-lg text-sm font-semibold flex items-center gap-2 border", statusClasses)}>
                        {status === 'running' && <Play size={14} className="fill-current" />}
                        {status === 'dead' && <AlertCircle size={14} />}
                        {status.toUpperCase()}
                    </div>
                    {isPaused && (
                        <div className="px-3 py-1.5 rounded-lg text-sm font-semibold flex items-center gap-2 border bg-amber-50 text-amber-600 border-amber-200">
                            <Pause size={14} />
                            MONITORING PAUSED
                        </div>
                    )}
                </div>
            </motion.div>

            {/* Info Cards */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className="bg-white rounded-xl border border-slate-200 p-4">
                    <p className="text-xs text-slate-500 uppercase font-medium mb-1">Image</p>
                    <p className="font-medium text-slate-800 truncate" title={container.image}>{container.image}</p>
                </div>
                <div className="bg-white rounded-xl border border-slate-200 p-4">
                    <p className="text-xs text-slate-500 uppercase font-medium mb-1">Uptime</p>
                    <p className="font-medium text-slate-800 flex items-center gap-1.5">
                        <Clock size={14} className="text-slate-400" />
                        {container.uptime || 'N/A'}
                    </p>
                </div>
                <div className="bg-white rounded-xl border border-slate-200 p-4">
                    <p className="text-xs text-slate-500 uppercase font-medium mb-1">Managed</p>
                    <p className="font-medium text-emerald-600 flex items-center gap-1.5">
                        <ShieldCheck size={14} />
                        Yes
                    </p>
                </div>
                <div className="bg-white rounded-xl border border-slate-200 p-4">
                    <p className="text-xs text-slate-500 uppercase font-medium mb-1">Monitoring</p>
                    <p className={cn("font-medium flex items-center gap-1.5", isPaused ? "text-amber-600" : "text-emerald-600")}>
                        {isPaused ? <ShieldOff size={14} /> : <Activity size={14} />}
                        {isPaused ? 'Paused' : 'Active'}
                    </p>
                </div>
            </div>

            {/* Watchdog State (if available) */}
            {container.watchdog && (
                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="bg-white rounded-xl border border-slate-200 p-4"
                >
                    <h3 className="text-sm font-semibold text-slate-700 mb-3 flex items-center gap-2">
                        <Activity size={16} className="text-indigo-500" />
                        Watchdog State
                    </h3>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                        <div>
                            <span className="text-slate-500">Failures:</span>
                            <span className={cn("ml-2 font-medium", container.watchdog.failures > 0 ? "text-rose-600" : "text-slate-800")}>
                                {container.watchdog.failures}
                            </span>
                        </div>
                        <div>
                            <span className="text-slate-500">Last Status:</span>
                            <span className="ml-2 font-medium text-slate-800">{container.watchdog.lastStatus || '-'}</span>
                        </div>
                        <div>
                            <span className="text-slate-500">Restart In Progress:</span>
                            <span className={cn("ml-2 font-medium", container.watchdog.restartInProgress ? "text-amber-600" : "text-slate-800")}>
                                {container.watchdog.restartInProgress ? 'Yes' : 'No'}
                            </span>
                        </div>
                        {container.watchdog.lastRestartAt && (
                            <div>
                                <span className="text-slate-500">Last Restart:</span>
                                <span className="ml-2 font-medium text-slate-800">
                                    {new Date(container.watchdog.lastRestartAt).toLocaleString()}
                                </span>
                            </div>
                        )}
                    </div>
                </motion.div>
            )}

            {/* Control Actions */}
            <div className="flex flex-wrap gap-3">
                <button
                    onClick={handleStart}
                    disabled={actionLoading !== null || container.status === 'running'}
                    className={cn(
                        "px-4 py-2 rounded-lg font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer",
                        container.status === 'running'
                            ? "bg-slate-100 text-slate-400 cursor-not-allowed"
                            : "bg-emerald-500 text-white hover:bg-emerald-600"
                    )}
                >
                    {actionLoading === 'start' ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
                    Start
                </button>
                <button
                    onClick={handleStop}
                    disabled={actionLoading !== null || container.status !== 'running'}
                    className={cn(
                        "px-4 py-2 rounded-lg font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer",
                        container.status !== 'running'
                            ? "bg-slate-100 text-slate-400 cursor-not-allowed"
                            : "bg-rose-500 text-white hover:bg-rose-600"
                    )}
                >
                    {actionLoading === 'stop' ? <Loader2 size={16} className="animate-spin" /> : <Square size={16} />}
                    Stop
                </button>
                <button
                    onClick={handleRestart}
                    disabled={actionLoading !== null}
                    className="px-4 py-2 rounded-lg bg-sky-500 text-white hover:bg-sky-600 font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer"
                >
                    {actionLoading === 'restart' ? <Loader2 size={16} className="animate-spin" /> : <RefreshCw size={16} />}
                    Restart
                </button>

                {/* Monitoring control */}
                <div className="border-l border-slate-200 pl-3 ml-1">
                    {isPaused ? (
                        <button
                            onClick={handleResumeMonitoring}
                            disabled={actionLoading !== null}
                            className="px-4 py-2 rounded-lg bg-amber-500 text-white hover:bg-amber-600 font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer"
                        >
                            {actionLoading === 'resume' ? <Loader2 size={16} className="animate-spin" /> : <CirclePlay size={16} />}
                            Resume Monitoring
                        </button>
                    ) : (
                        <button
                            onClick={handlePauseMonitoring}
                            disabled={actionLoading !== null}
                            className="px-4 py-2 rounded-lg bg-slate-500 text-white hover:bg-slate-600 font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer"
                        >
                            {actionLoading === 'pause' ? <Loader2 size={16} className="animate-spin" /> : <Pause size={16} />}
                            Pause Monitoring
                        </button>
                    )}
                </div>
            </div>

            {/* Logs */}
            <div>
                <h2 className="text-lg font-bold text-slate-800 mb-3">Logs</h2>
                <div className="h-[400px]">
                    <LogViewer containerName={name!} />
                </div>
            </div>
        </div>
    );
}
