import { useEffect, useState } from 'react';
import { getEvents } from '@/api/client';
import { WatchdogEvent } from '@/types';
import { Loader2, Activity, Info } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn } from '@/utils';

export function EventsLog() {
    const [events, setEvents] = useState<WatchdogEvent[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(function fetchEvents() {
        async function load() {
            try {
                const data = await getEvents();
                // data.events depends on actual API response, assuming { events: [...] } or just [...]
                // Based on client.ts return type inference it returns { events: any[] }
                setEvents(data.events || []);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load events');
            } finally {
                setLoading(false);
            }
        }
        load();
        const interval = setInterval(load, 5000);
        return function cleanup() { clearInterval(interval); };
    }, []);

    if (loading && events.length === 0) {
        return (
            <div className="flex items-center justify-center h-64 text-slate-400">
                <Loader2 className="animate-spin mr-2" />
                Loading events...
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-6 text-center text-rose-500 bg-rose-50 rounded-xl border border-rose-200">
                <p>Error loading events: {error}</p>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold text-slate-800 flex items-center gap-2">
                    <Activity className="text-indigo-500" />
                    System Events
                </h1>
                <span className="text-sm text-slate-500 bg-slate-100 px-3 py-1 rounded-full">
                    Last {events.length} events
                </span>
            </div>

            <div className="relative border-l-2 border-slate-200 ml-4 space-y-8 py-2">
                {events.length === 0 ? (
                    <div className="pl-6 text-slate-400 italic">No events recorded yet.</div>
                ) : (
                    events.map((evt, idx) => (
                        <motion.div
                            key={idx}
                            initial={{ opacity: 0, x: -10 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: idx * 0.05 }}
                            className="relative pl-6"
                        >
                            <span className={cn(
                                "absolute -left-[9px] top-1 h-4 w-4 rounded-full border-2 border-white shadow-sm flex items-center justify-center",
                                evt.level === 'error' ? "bg-rose-500" :
                                    evt.level === 'warn' ? "bg-amber-500" : "bg-sky-500"
                            )}>
                            </span>

                            <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm hover:shadow-md transition-shadow">
                                <div className="flex justify-between items-start mb-1">
                                    <span className={cn(
                                        "text-xs font-bold uppercase tracking-wider px-2 py-0.5 rounded",
                                        evt.level === 'error' ? "bg-rose-50 text-rose-600" :
                                            evt.level === 'warn' ? "bg-amber-50 text-amber-600" : "bg-sky-50 text-sky-600"
                                    )}>
                                        {evt.level}
                                    </span>
                                    <span className="text-xs text-slate-400 font-mono">
                                        {new Date(evt.timestamp).toLocaleString()}
                                    </span>
                                </div>
                                <p className="text-slate-800 font-medium mt-2">
                                    {evt.message}
                                </p>
                                <p className="text-xs text-slate-400 mt-2 flex items-center gap-1">
                                    <Info size={12} />
                                    Source: {evt.source}
                                </p>
                            </div>
                        </motion.div>
                    ))
                )}
            </div>
        </div>
    );
}
