import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { LayoutDashboard, Server, Box, Activity, Settings } from 'lucide-react';
import { RouteItem } from './types';

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

export function getStatusColor(status?: string): string {
    switch (status?.toLowerCase() || 'unknown') {
        case 'running': return 'text-emerald-500 bg-emerald-50 border-emerald-200';
        case 'dead':
        case 'exited': return 'text-rose-500 bg-rose-50 border-rose-200';
        case 'paused': return 'text-amber-500 bg-amber-50 border-amber-200';
        default: return 'text-slate-500 bg-slate-50 border-slate-200';
    }
}

export const routes: RouteItem[] = [
    { path: '/', label: 'Dashboard', icon: LayoutDashboard },
    { path: '/containers', label: 'Managed Targets', icon: Server },
    { path: '/docker', label: 'Docker Inventory', icon: Box },
    { path: '/events', label: 'Events', icon: Activity },
    { path: '/settings', label: 'Settings', icon: Settings },
];

