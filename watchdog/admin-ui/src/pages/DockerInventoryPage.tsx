import { useEffect, useState } from 'react';
import { getContainers, setContainerManaged, getTargets } from '@/api/client';
import { ContainerInfo } from '@/types';
import { cn, getStatusColor } from '@/utils';
import {
    Loader2,
    Box,
    ShieldCheck,
    ShieldOff,
    Play,
    Square,
    AlertCircle,
    Search,
    Filter,
    RefreshCw
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { useToast } from '@/stores/toastStore';

type FilterType = 'all' | 'managed' | 'unmanaged' | 'running' | 'stopped';

export function DockerInventoryPage() {
    const [containers, setContainers] = useState<ContainerInfo[]>([]);
    const [managedNames, setManagedNames] = useState<Set<string>>(new Set());
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [search, setSearch] = useState('');
    const [filter, setFilter] = useState<FilterType>('all');
    const [actionLoading, setActionLoading] = useState<string | null>(null);
    const { addToast } = useToast();

    async function loadData() {
        try {
            setError(null);
            const [containersData, targetsData] = await Promise.all([
                getContainers(),
                getTargets()
            ]);
            setContainers(containersData.containers || []);
            // Build set of managed container names
            const managed = new Set((targetsData.targets || []).map(t => t.name));
            setManagedNames(managed);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load containers');
        } finally {
            setLoading(false);
        }
    }

    useEffect(function fetchData() {
        loadData();
        const interval = setInterval(loadData, 10000);
        return function cleanup() { clearInterval(interval); };
    }, []);

    async function toggleManaged(container: ContainerInfo) {
        const isCurrentlyManaged = managedNames.has(container.name);
        const newManaged = !isCurrentlyManaged;
        const action = newManaged ? '관리대상 추가' : '관리대상 제외';

        if (!confirm(`${container.name}을(를) ${action}하시겠습니까?`)) {
            return;
        }

        setActionLoading(container.name);
        try {
            await setContainerManaged(container.name, newManaged, `admin_toggle_${newManaged ? 'enable' : 'disable'}`);
            // Update local state
            setManagedNames(prev => {
                const next = new Set(prev);
                if (newManaged) {
                    next.add(container.name);
                } else {
                    next.delete(container.name);
                }
                return next;
            });
            addToast('success', `${container.name}: ${action} 완료`);
        } catch (err) {
            addToast('error', `${action} 실패: ${err instanceof Error ? err.message : 'Unknown'}`);
        } finally {
            setActionLoading(null);
        }
    }

    // Filter and search
    const filteredContainers = containers.filter(c => {
        // Search filter
        if (search && !c.name.toLowerCase().includes(search.toLowerCase()) &&
            !c.image.toLowerCase().includes(search.toLowerCase())) {
            return false;
        }
        // Status filter
        const isManaged = managedNames.has(c.name);
        switch (filter) {
            case 'managed':
                return isManaged;
            case 'unmanaged':
                return !isManaged;
            case 'running':
                return c.status === 'running';
            case 'stopped':
                return c.status !== 'running';
            default:
                return true;
        }
    });

    if (loading && containers.length === 0) {
        return (
            <div className="flex items-center justify-center h-64 text-slate-400">
                <Loader2 className="animate-spin mr-2" />
                Loading Docker containers...
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-6 text-center text-rose-500 bg-rose-50 rounded-xl border border-rose-200">
                <AlertCircle className="mx-auto w-10 h-10 mb-2 opacity-50" />
                <p className="font-medium">Failed to load containers</p>
                <p className="text-sm mt-1">{error}</p>
                <button
                    onClick={loadData}
                    className="mt-4 px-4 py-2 bg-rose-500 text-white rounded-lg hover:bg-rose-600 transition-colors"
                >
                    Retry
                </button>
            </div>
        );
    }

    const managedCount = containers.filter(c => managedNames.has(c.name)).length;
    const runningCount = containers.filter(c => c.status === 'running').length;

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-800 flex items-center gap-2">
                        <Box className="text-indigo-500" />
                        Docker Inventory
                    </h1>
                    <p className="text-slate-500 text-sm mt-1">
                        전체 {containers.length}개 컨테이너 • {managedCount}개 관리중 • {runningCount}개 실행중
                    </p>
                </div>
                <button
                    onClick={loadData}
                    className="px-4 py-2 bg-white border border-slate-200 rounded-lg text-sm font-medium text-slate-700 hover:bg-slate-50 flex items-center gap-2 shadow-sm"
                >
                    <RefreshCw size={14} />
                    Refresh
                </button>
            </div>

            {/* Filters */}
            <div className="flex flex-col md:flex-row gap-4">
                <div className="relative flex-1">
                    <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                    <input
                        type="text"
                        placeholder="Search containers..."
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="w-full pl-9 pr-4 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-200 focus:border-indigo-400"
                    />
                </div>
                <div className="flex items-center gap-2">
                    <Filter size={16} className="text-slate-400" />
                    {(['all', 'managed', 'unmanaged', 'running', 'stopped'] as FilterType[]).map(f => (
                        <button
                            key={f}
                            onClick={() => setFilter(f)}
                            className={cn(
                                "px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
                                filter === f
                                    ? "bg-indigo-500 text-white"
                                    : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                            )}
                        >
                            {f.charAt(0).toUpperCase() + f.slice(1)}
                        </button>
                    ))}
                </div>
            </div>

            {/* Container List */}
            <div className="bg-white rounded-2xl border border-slate-200 overflow-hidden shadow-sm">
                <div className="overflow-x-auto">
                    <table className="w-full">
                        <thead>
                            <tr className="bg-slate-50 border-b border-slate-200">
                                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-4 py-3">Container</th>
                                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-4 py-3">Image</th>
                                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-4 py-3">Status</th>
                                <th className="text-left text-xs font-semibold text-slate-500 uppercase tracking-wider px-4 py-3">Uptime</th>
                                <th className="text-center text-xs font-semibold text-slate-500 uppercase tracking-wider px-4 py-3">Managed</th>
                                <th className="text-center text-xs font-semibold text-slate-500 uppercase tracking-wider px-4 py-3">Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            <AnimatePresence>
                                {filteredContainers.length === 0 ? (
                                    <tr>
                                        <td colSpan={6} className="text-center py-8 text-slate-400">
                                            No containers match your filters.
                                        </td>
                                    </tr>
                                ) : (
                                    filteredContainers.map((c, idx) => {
                                        const isManaged = managedNames.has(c.name);
                                        const statusClasses = getStatusColor(c.status);

                                        return (
                                            <motion.tr
                                                key={c.id}
                                                initial={{ opacity: 0, y: 10 }}
                                                animate={{ opacity: 1, y: 0 }}
                                                exit={{ opacity: 0 }}
                                                transition={{ delay: idx * 0.02 }}
                                                className="border-b border-slate-100 hover:bg-slate-50/50 transition-colors"
                                            >
                                                <td className="px-4 py-3">
                                                    <div className="flex items-center gap-2">
                                                        <Box size={16} className="text-slate-400" />
                                                        <span className="font-medium text-slate-800">{c.name}</span>
                                                    </div>
                                                    <span className="text-xs text-slate-400 font-mono">{c.id.substring(0, 12)}</span>
                                                </td>
                                                <td className="px-4 py-3">
                                                    <span className="text-sm text-slate-600 truncate max-w-xs block" title={c.image}>
                                                        {c.image.length > 40 ? c.image.substring(0, 37) + '...' : c.image}
                                                    </span>
                                                </td>
                                                <td className="px-4 py-3">
                                                    <span className={cn("px-2 py-1 rounded-md text-xs font-semibold inline-flex items-center gap-1", statusClasses)}>
                                                        {c.status === 'running' ? <Play size={10} className="fill-current" /> : <Square size={10} />}
                                                        {c.status}
                                                    </span>
                                                </td>
                                                <td className="px-4 py-3 text-sm text-slate-600">
                                                    {c.uptime || '-'}
                                                </td>
                                                <td className="px-4 py-3 text-center">
                                                    {isManaged ? (
                                                        <span className="inline-flex items-center gap-1 px-2 py-1 bg-emerald-50 text-emerald-600 rounded-md text-xs font-semibold">
                                                            <ShieldCheck size={12} />
                                                            Managed
                                                        </span>
                                                    ) : (
                                                        <span className="inline-flex items-center gap-1 px-2 py-1 bg-slate-100 text-slate-500 rounded-md text-xs font-medium">
                                                            <ShieldOff size={12} />
                                                            Unmanaged
                                                        </span>
                                                    )}
                                                </td>
                                                <td className="px-4 py-3 text-center">
                                                    <button
                                                        onClick={() => toggleManaged(c)}
                                                        disabled={actionLoading !== null}
                                                        className={cn(
                                                            "px-3 py-1.5 rounded-lg text-xs font-medium transition-colors disabled:opacity-50",
                                                            isManaged
                                                                ? "bg-rose-50 text-rose-600 hover:bg-rose-100 border border-rose-200"
                                                                : "bg-emerald-50 text-emerald-600 hover:bg-emerald-100 border border-emerald-200"
                                                        )}
                                                    >
                                                        {actionLoading === c.name ? (
                                                            <Loader2 size={12} className="animate-spin inline" />
                                                        ) : isManaged ? (
                                                            'Remove'
                                                        ) : (
                                                            'Add'
                                                        )}
                                                    </button>
                                                </td>
                                            </motion.tr>
                                        );
                                    })
                                )}
                            </AnimatePresence>
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Info Box */}
            <div className="p-4 bg-amber-50 border border-amber-200 rounded-xl">
                <h4 className="font-semibold text-amber-800 flex items-center gap-2">
                    <AlertCircle size={16} />
                    관리대상 설정 안내
                </h4>
                <p className="text-sm text-amber-700 mt-1">
                    • <strong>Managed</strong> 컨테이너는 watchdog이 헬스체크 및 자동 재시작을 수행합니다.<br />
                    • 변경 시 <code className="bg-amber-100 px-1 rounded">config.json</code> 파일이 자동으로 업데이트됩니다.<br />
                    • <code className="bg-amber-100 px-1 rounded">WATCHDOG_CONFIG_PATH</code> 환경변수가 설정되어 있어야 합니다.
                </p>
            </div>
        </div>
    );
}
