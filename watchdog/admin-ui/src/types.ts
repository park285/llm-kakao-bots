import { LucideIcon } from 'lucide-react';

export interface RouteItem {
    path: string;
    label: string;
    icon?: LucideIcon;
}

export type ContainerStatus = 'running' | 'paused' | 'restarting' | 'dead' | 'exited' | 'unknown';

export interface WatchdogState {
    failures: number;
    lastStatus: string;
    lastCheckedAt?: string;
    cooldownUntil?: string;
    restartInProgress: boolean;
    lastRestartAt?: string;
    lastRestartBy?: string;
    lastRestartRequestedBy?: string;
    lastRestartReason?: string;
    lastRestartResult?: string;
    lastRestartError?: string;
}

export interface DockerState {
    found: boolean;
    id?: string;
    image?: string;
    state?: string;
    status?: string;
    health?: string;
    startedAt?: string;
    finishedAt?: string;
    exitCode?: number;
    restartCount?: number;
    uptimeSec?: number;
}

export interface ContainerInfo {
    name: string;
    id: string;
    image: string;
    status: ContainerStatus;
    state: string;
    managed: boolean;
    uptime: string;
    // Extended fields from TargetStatus
    monitoringPaused?: boolean;
    watchdog?: WatchdogState;
    docker?: DockerState;
}

export interface TargetStatus {
    name: string;
    monitoringPaused: boolean;
    watchdog: WatchdogState;
    docker: DockerState;
}

export interface ApiResponse<T> {
    data?: T;
    error?: {
        code: string;
        message: string;
    };
}

export interface WatchdogEvent {
    id: string;
    timestamp: string;
    level: 'info' | 'warn' | 'error';
    message: string;
    source: string;
}

export interface EventsResponse {
    events: WatchdogEvent[];
}
