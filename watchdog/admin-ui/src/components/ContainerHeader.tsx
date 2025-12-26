import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ArrowLeft, Play, AlertCircle, Pause, Box } from 'lucide-react';
import { cn, getStatusColor } from '@/utils';
import { ContainerInfo } from '@/types';

type ContainerHeaderProps = {
    container: ContainerInfo;
    isPaused: boolean;
}

export function ContainerHeader({ container, isPaused }: ContainerHeaderProps) {
    const status = container.status || 'unknown';
    const statusClasses = getStatusColor(status);

    return (
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
    );
}
