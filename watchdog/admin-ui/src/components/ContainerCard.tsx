import { ContainerInfo } from '@/types';
import { getStatusColor, cn } from '@/utils';
import { Box, Play, Square, RefreshCw, AlertCircle, Clock } from 'lucide-react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';

interface ContainerCardProps {
    container: ContainerInfo;
}

export function ContainerCard({ container }: ContainerCardProps) {
    const status = container.status || 'unknown';
    const statusColorClass = getStatusColor(status);
    const containerId = container.id || '';
    const containerName = container.name || 'unnamed';
    const containerImage = container.image || 'unknown';

    return (
        <Link to={`/containers/${containerName}`} className="block group">
            <motion.div
                whileHover={{ y: -4 }}
                className="relative bg-white rounded-2xl p-5 border border-slate-200 shadow-sm hover:shadow-md transition-all h-full"
            >
                <div className="flex justify-between items-start mb-4">
                    <div className="flex items-center gap-3">
                        <div className="p-2.5 bg-indigo-50 rounded-xl text-indigo-600 group-hover:bg-indigo-100 transition-colors">
                            <Box size={20} />
                        </div>
                        <div>
                            <h3 className="font-bold text-slate-800 text-lg leading-tight group-hover:text-indigo-600 transition-colors">
                                {containerName}
                            </h3>
                            <p className="text-xs text-slate-400 font-mono mt-1">
                                {containerId.substring(0, 12) || '-'}
                            </p>
                        </div>
                    </div>

                    <div className={cn("px-2.5 py-1 rounded-lg text-xs font-semibold flex items-center gap-1.5 border", statusColorClass)}>
                        {status === 'running' && <Play size={12} className="fill-current" />}
                        {status === 'paused' && <Square size={12} className="fill-current" />}
                        {status === 'restarting' && <RefreshCw size={12} className="animate-spin" />}
                        {status === 'dead' && <AlertCircle size={12} />}
                        {status.toUpperCase()}
                    </div>
                </div>

                <div className="space-y-3">
                    <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-500">Image</span>
                        <span className="font-medium text-slate-700 truncate max-w-[120px]" title={containerImage}>
                            {containerImage}
                        </span>
                    </div>

                    <div className="pt-3 border-t border-slate-100 flex items-center gap-2 text-xs text-slate-500">
                        <Clock size={14} />
                        <span>Up {container.uptime || '0s'}</span>
                    </div>
                </div>
            </motion.div>
        </Link>
    );
}
