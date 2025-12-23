import { useEffect, useState } from 'react';
import { getTargets, getWatchdogStatus } from '@/api/client';
import { ContainerInfo } from '@/types';
import { ContainerCard } from '@/components/ContainerCard';
import { StatCard } from '@/components/ui/StatCard';
import { Activity, Server, ShieldCheck, Loader2, AlertTriangle } from 'lucide-react';
import { motion } from 'framer-motion';

export function Dashboard() {
    const [targets, setTargets] = useState<ContainerInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [uptime, setUptime] = useState<string>('-');

    async function load() {
        setLoading(true);
        setError(null);
        try {
            const [statusData, targetsData] = await Promise.all([
                getWatchdogStatus(),
                getTargets()
            ]);
            setUptime(statusData.uptime);
            setTargets(targetsData.targets || []);
        } catch (err) {
            if (err instanceof Error) {
                if (err.message.includes('500')) {
                    setError('Backend error (500). Check if CF Access env vars are set.');
                } else {
                    setError(err.message);
                }
            } else {
                setError('Unknown error occurred');
            }
        } finally {
            setLoading(false);
        }
    }

    useEffect(function fetchData() {
        load();
        const interval = setInterval(() => {
            getWatchdogStatus().then(s => setUptime(s.uptime)).catch(() => { });
            getTargets().then(t => setTargets(t.targets || [])).catch(() => { });
        }, 5000);
        return function cleanup() { clearInterval(interval); };
    }, []);

    if (loading && targets.length === 0) {
        return (
            <div className="flex justify-center items-center h-64 text-slate-400">
                <Loader2 className="animate-spin mr-2" />
                Loading Dashboard...
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-8 max-w-2xl mx-auto mt-10 bg-white border border-rose-200 rounded-2xl shadow-sm text-center">
                <div className="w-16 h-16 bg-rose-50 rounded-full flex items-center justify-center mx-auto mb-4">
                    <Activity className="w-8 h-8 text-rose-500" />
                </div>
                <h3 className="text-xl font-bold text-slate-800 mb-2">Connection Issue</h3>
                <p className="text-slate-600 mb-6">{error}</p>
                <button
                    onClick={load}
                    className="px-6 py-2.5 bg-indigo-600 text-white rounded-xl font-medium hover:bg-indigo-700 transition-colors shadow-sm shadow-indigo-200"
                >
                    Retry
                </button>
            </div>
        );
    }

    const runningCount = targets.filter(c => c.status === 'running').length;
    const issueCount = targets.filter(c => c.status !== 'running').length;

    return (
        <div className="space-y-8">
            {/* Welcome Banner */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                className="relative overflow-hidden rounded-3xl bg-white border border-slate-100 p-8 shadow-sm"
            >
                <div className="absolute top-0 right-0 w-96 h-96 bg-indigo-50 rounded-full blur-3xl opacity-60 -mr-20 -mt-20 pointer-events-none"></div>
                <div className="relative z-10 flex flex-col md:flex-row items-center justify-between gap-8">
                    <div className="max-w-xl">
                        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-50 border border-emerald-100 text-emerald-600 text-xs font-semibold mb-4">
                            <span className="relative flex h-2 w-2">
                                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                            </span>
                            System Online
                        </div>
                        <h1 className="text-3xl font-bold text-slate-800 tracking-tight">
                            Dashboard
                        </h1>
                        <p className="text-slate-500 mt-2">
                            Uptime: <span className="font-mono bg-slate-100 px-1.5 py-0.5 rounded text-slate-700">{uptime}</span>
                        </p>
                    </div>
                    <div className="hidden md:flex items-center justify-center w-16 h-16 bg-white rounded-2xl shadow-sm border border-slate-100">
                        <ShieldCheck className="w-8 h-8 text-indigo-500" />
                    </div>
                </div>
            </motion.div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <StatCard label="Managed Targets" value={targets.length} icon={<Server size={24} />} variant="indigo" />
                <StatCard label="Running" value={runningCount} icon={<Activity size={24} />} variant="green" />
                <StatCard label="Issues" value={issueCount} icon={<Activity size={24} />} variant={issueCount > 0 ? "rose" : "blue"} />
            </div>

            {targets.length === 0 ? (
                <div className="p-8 bg-amber-50 border border-amber-200 rounded-2xl text-center">
                    <AlertTriangle className="w-10 h-10 text-amber-500 mx-auto mb-3" />
                    <h3 className="text-lg font-bold text-slate-800 mb-2">No Managed Targets</h3>
                    <p className="text-slate-600 text-sm">
                        No containers are configured for monitoring. Add containers via <code className="bg-slate-200 px-1 rounded">WATCHDOG_CONTAINERS</code> env var or config JSON.
                    </p>
                </div>
            ) : (
                <div>
                    <h2 className="text-xl font-bold text-slate-800 mb-6">Managed Containers</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
                        {targets.map(c => (
                            <ContainerCard key={c.name} container={c} />
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
