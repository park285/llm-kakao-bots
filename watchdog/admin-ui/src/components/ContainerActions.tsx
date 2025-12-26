import { Loader2, Play, Square, RefreshCw, Pause, CirclePlay } from 'lucide-react';
import { cn } from '@/utils';

type ContainerActionsProps = {
    status: string;
    isPaused: boolean;
    actionLoading: string | null;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onPauseMonitoring: () => void;
    onResumeMonitoring: () => void;
}

export function ContainerActions({
    status,
    isPaused,
    actionLoading,
    onStart,
    onStop,
    onRestart,
    onPauseMonitoring,
    onResumeMonitoring,
}: ContainerActionsProps) {
    return (
        <div className="flex flex-wrap gap-3">
            <button
                onClick={onStart}
                disabled={actionLoading !== null || status === 'running'}
                className={cn(
                    "px-4 py-2 rounded-lg font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer",
                    status === 'running'
                        ? "bg-slate-100 text-slate-400 cursor-not-allowed"
                        : "bg-emerald-500 text-white hover:bg-emerald-600"
                )}
            >
                {actionLoading === 'start' ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
                Start
            </button>
            <button
                onClick={onStop}
                disabled={actionLoading !== null || status !== 'running'}
                className={cn(
                    "px-4 py-2 rounded-lg font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer",
                    status !== 'running'
                        ? "bg-slate-100 text-slate-400 cursor-not-allowed"
                        : "bg-rose-500 text-white hover:bg-rose-600"
                )}
            >
                {actionLoading === 'stop' ? <Loader2 size={16} className="animate-spin" /> : <Square size={16} />}
                Stop
            </button>
            <button
                onClick={onRestart}
                disabled={actionLoading !== null}
                className="px-4 py-2 rounded-lg bg-sky-500 text-white hover:bg-sky-600 font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer"
            >
                {actionLoading === 'restart' ? <Loader2 size={16} className="animate-spin" /> : <RefreshCw size={16} />}
                Restart
            </button>

            <div className="border-l border-slate-200 pl-3 ml-1">
                {isPaused ? (
                    <button
                        onClick={onResumeMonitoring}
                        disabled={actionLoading !== null}
                        className="px-4 py-2 rounded-lg bg-amber-500 text-white hover:bg-amber-600 font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer"
                    >
                        {actionLoading === 'resume' ? <Loader2 size={16} className="animate-spin" /> : <CirclePlay size={16} />}
                        Resume Monitoring
                    </button>
                ) : (
                    <button
                        onClick={onPauseMonitoring}
                        disabled={actionLoading !== null}
                        className="px-4 py-2 rounded-lg bg-slate-500 text-white hover:bg-slate-600 font-medium text-sm flex items-center gap-2 transition-colors cursor-pointer"
                    >
                        {actionLoading === 'pause' ? <Loader2 size={16} className="animate-spin" /> : <Pause size={16} />}
                        Pause Monitoring
                    </button>
                )}
            </div>
        </div>
    );
}
