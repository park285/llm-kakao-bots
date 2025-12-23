import { useEffect, useRef, useState } from 'react';
import { getLogStreamUrl, getContainerLogs } from '@/api/client';
import { cn } from '@/utils';
import { Play, Pause, Trash2, Loader2 } from 'lucide-react';

interface LogViewerProps {
    containerName: string;
}

export function LogViewer({ containerName }: LogViewerProps) {
    const [logs, setLogs] = useState<string[]>([]);
    const [isFollowing, setIsFollowing] = useState(true);
    const [isConnected, setIsConnected] = useState(false);
    const [isLoading, setIsLoading] = useState(true);
    const logEndRef = useRef<HTMLDivElement>(null);
    const eventSourceRef = useRef<EventSource | null>(null);

    // Scroll to bottom when logs update if following
    useEffect(function scrollWhenFollowing() {
        if (isFollowing && logEndRef.current) {
            logEndRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [logs, isFollowing]);

    // Initial log fetch and SSE connection
    useEffect(function setupLogStream() {
        let isMounted = true;

        async function init() {
            setIsLoading(true);
            try {
                // Fetch initial logs
                const initialLogs = await getContainerLogs(containerName, 100);
                if (isMounted) {
                    setLogs(initialLogs.split('\n').filter(Boolean));
                }
            } catch (err) {
                console.error('Failed to fetch initial logs:', err);
                if (isMounted) {
                    setLogs([`[Error] Failed to load initial logs: ${err instanceof Error ? err.message : 'Unknown'}`]);
                }
            } finally {
                if (isMounted) setIsLoading(false);
            }

            // Setup SSE
            const url = getLogStreamUrl(containerName, 0); // tail=0 for stream, we already have initial
            const es = new EventSource(url);
            eventSourceRef.current = es;

            es.onopen = function handleOpen() {
                if (isMounted) setIsConnected(true);
            };

            es.onmessage = function handleMessage(event: MessageEvent) {
                if (isMounted && event.data) {
                    try {
                        const parsed = JSON.parse(event.data as string) as { log?: string };
                        if (parsed.log) {
                            setLogs(prev => [...prev.slice(-500), parsed.log!]); // Keep last 500 lines
                        }
                    } catch {
                        // Raw text fallback
                        setLogs(prev => [...prev.slice(-500), event.data as string]);
                    }
                }
            };

            es.onerror = function handleError() {
                if (isMounted) setIsConnected(false);
                es.close();
            };
        }

        init();

        return function cleanup() {
            isMounted = false;
            eventSourceRef.current?.close();
        };
    }, [containerName]);

    function handleClear() {
        setLogs([]);
    }

    function toggleFollow() {
        setIsFollowing(prev => !prev);
    }

    return (
        <div className="flex flex-col h-full bg-slate-900 rounded-xl border border-slate-700 overflow-hidden">
            {/* Toolbar */}
            <div className="flex items-center justify-between px-4 py-2 bg-slate-800 border-b border-slate-700">
                <div className="flex items-center gap-3">
                    <span className={cn(
                        "h-2.5 w-2.5 rounded-full",
                        isConnected ? "bg-emerald-400 animate-pulse" : "bg-rose-400"
                    )} />
                    <span className="text-xs text-slate-400 font-mono">
                        {isConnected ? 'Live' : 'Disconnected'}
                    </span>
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={toggleFollow}
                        className={cn(
                            "p-1.5 rounded-md text-xs flex items-center gap-1",
                            isFollowing ? "bg-sky-600 text-white" : "bg-slate-700 text-slate-300 hover:bg-slate-600"
                        )}
                        title={isFollowing ? 'Pause auto-scroll' : 'Resume auto-scroll'}
                    >
                        {isFollowing ? <Pause size={14} /> : <Play size={14} />}
                        Follow
                    </button>
                    <button
                        onClick={handleClear}
                        className="p-1.5 rounded-md bg-slate-700 text-slate-300 hover:bg-slate-600 text-xs flex items-center gap-1"
                        title="Clear logs"
                    >
                        <Trash2 size={14} />
                        Clear
                    </button>
                </div>
            </div>

            {/* Log Area */}
            <div className="flex-1 overflow-y-auto p-4 font-mono text-xs leading-relaxed text-slate-300 scrollbar-thin scrollbar-thumb-slate-700">
                {isLoading ? (
                    <div className="flex items-center justify-center h-full text-slate-500">
                        <Loader2 className="animate-spin mr-2" size={16} />
                        Loading logs...
                    </div>
                ) : logs.length === 0 ? (
                    <div className="text-slate-500 text-center py-8">No logs available.</div>
                ) : (
                    logs.map((line, idx) => (
                        <div key={idx} className="whitespace-pre-wrap break-all hover:bg-slate-800/50 px-1 -mx-1 rounded">
                            {line}
                        </div>
                    ))
                )}
                <div ref={logEndRef} />
            </div>
        </div>
    );
}
