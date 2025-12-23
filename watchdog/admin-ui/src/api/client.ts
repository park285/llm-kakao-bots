import { ApiResponse, ContainerInfo, ContainerStatus, DockerState, TargetStatus } from '../types';

const API_BASE = '/admin/api/v1';

type TargetsResponse = { generatedAt: string; targets: TargetStatus[] };

// Helper to handle API responses
async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const headers = new Headers(options?.headers);
    // Development mode: skip CF Access auth
    const separator = endpoint.includes('?') ? '&' : '?';
    const devEndpoint = `${endpoint}${separator}skip_auth=true`;

    const res = await fetch(`${API_BASE}${devEndpoint}`, {
        ...options,
        headers,
    });

    if (!res.ok) {
        throw new Error(`API call failed: ${res.status} ${res.statusText}`);
    }

    const json: ApiResponse<T> = await res.json();
    if (json.error) {
        throw new Error(json.error.message);
    }

    // Handle cases where data is wrapped in 'data' field or returned directly
    return (json.data ?? json) as T;
}

// Format uptime seconds to human-readable string
export function formatUptime(uptimeSec: number): string {
    if (uptimeSec < 60) {
        return `${uptimeSec}s`;
    }
    if (uptimeSec < 3600) {
        const m = Math.floor(uptimeSec / 60);
        const s = uptimeSec % 60;
        return s > 0 ? `${m}m ${s}s` : `${m}m`;
    }
    if (uptimeSec < 86400) {
        const h = Math.floor(uptimeSec / 3600);
        const m = Math.floor((uptimeSec % 3600) / 60);
        return m > 0 ? `${h}h ${m}m` : `${h}h`;
    }
    const d = Math.floor(uptimeSec / 86400);
    const h = Math.floor((uptimeSec % 86400) / 3600);
    return h > 0 ? `${d}d ${h}h` : `${d}d`;
}

function normalizeContainerStatus(raw?: string): ContainerStatus {
    switch (raw) {
        case 'running':
        case 'paused':
        case 'restarting':
        case 'dead':
        case 'exited':
            return raw;
        default:
            return 'unknown';
    }
}

function buildContainerInfoFromTarget(target: TargetStatus): ContainerInfo {
    const docker: DockerState = target.docker ?? { found: false };
    const uptimeSec = typeof docker.uptimeSec === 'number' ? docker.uptimeSec : 0;
    const status = normalizeContainerStatus(docker.state);

    return {
        name: target.name,
        id: docker.id ?? '',
        image: docker.image ?? '',
        status,
        state: docker.state ?? '',
        managed: true,
        uptime: formatUptime(uptimeSec),
        monitoringPaused: target.monitoringPaused ?? false,
        watchdog: target.watchdog,
        docker,
    };
}

export interface WatchdogStatus {
    startedAt: string;
    uptimeSec: number;
    uptime: string;
    enabled: boolean;
    configSource: string;
    configPath: string;
    containers: string[];
    intervalSec: number;
    maxFailures: number;
    cooldownSec: number;
    restartTimeoutSec: number;
    dockerSocket: string;
    useEvents: boolean;
    statusReportSec: number;
    verbose: boolean;
}

export async function getWatchdogStatus(): Promise<WatchdogStatus> {
    const data = await request<WatchdogStatus>('/watchdog/status');
    // Add formatted uptime
    return {
        ...data,
        uptime: formatUptime(data.uptimeSec),
    };
}

export async function getContainers(): Promise<{ generatedAt: string; containers: ContainerInfo[] }> {
    return request('/docker/containers');
}

export async function getTargets(): Promise<{ generatedAt: string; targets: ContainerInfo[] }> {
    const data = await request<TargetsResponse>('/targets');
    return {
        generatedAt: data.generatedAt,
        targets: (data.targets || []).map(buildContainerInfoFromTarget),
    };
}

export async function restartContainer(name: string, force = false): Promise<void> {
    await request(`/targets/${name}/restart`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ force }),
    });
}

export async function startContainer(name: string): Promise<void> {
    await request(`/targets/${name}/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
    });
}

export async function stopContainer(name: string, timeoutSeconds = 10): Promise<void> {
    await request(`/targets/${name}/stop`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ timeoutSeconds }),
    });
}

export async function pauseMonitoring(name: string): Promise<void> {
    await request(`/targets/${name}/pause`, {
        method: 'POST',
    });
}

export async function resumeMonitoring(name: string): Promise<void> {
    await request(`/targets/${name}/resume`, {
        method: 'POST',
    });
}

export async function getContainerLogs(name: string, tail = 200): Promise<string> {
    const res = await fetch(`${API_BASE}/targets/${name}/logs?tail=${tail}&timestamps=true&skip_auth=true`);
    if (!res.ok) {
        throw new Error(`Failed to fetch logs: ${res.status}`);
    }
    return res.text();
}

export function getLogStreamUrl(name: string, tail = 200): string {
    return `${API_BASE}/targets/${name}/logs/stream?tail=${tail}&skip_auth=true`;
}

export async function getTargetDetails(name: string): Promise<ContainerInfo> {
    const data = await request<TargetStatus>(`/targets/${name}`);
    return buildContainerInfoFromTarget(data);
}

export async function getEvents(limit = 200): Promise<{ events: any[] }> {
    return request(`/events?limit=${limit}`);
}

// Fetch a specific container's details.
export async function getContainerByName(name: string): Promise<ContainerInfo | null> {
    try {
        const data = await request<TargetStatus>(`/targets/${name}`);
        return buildContainerInfoFromTarget(data);
    } catch {
        const data = await getContainers();
        const container = data.containers.find(c => c.name === name);
        return container ?? null;
    }
}

// Reload config from disk
export async function reloadConfig(): Promise<{ appliedFields: string[]; requiresRestartFields: string[] }> {
    return request('/watchdog/reload-config', {
        method: 'POST',
    });
}

// Trigger immediate health check
export async function triggerCheckNow(): Promise<{ status: string }> {
    return request('/watchdog/check-now', {
        method: 'POST',
    });
}

// Set container managed status
export async function setContainerManaged(name: string, managed: boolean, reason?: string): Promise<{ status: string; container: string; managed: boolean }> {
    return request(`/targets/${name}/managed`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ managed, reason }),
    });
}

// Toggle global watchdog enabled state
export async function setWatchdogEnabled(enabled: boolean, reason?: string): Promise<{ status: string; enabled: boolean }> {
    return request('/watchdog/enabled', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled, reason }),
    });
}
