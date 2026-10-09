import type { Agent, AgentProfile, Workspace, World } from '../domain/model';
import { getTauriApiBase, initTauriApiPort, isTauri } from '../tauri/runtime';
import { textOf } from './fields';

let goApiBase: null | string = null;

function browserApiBase(): null | string {
    if (process.env.NEXT_PUBLIC_API_URL !== undefined && process.env.NEXT_PUBLIC_API_URL !== '') {
        return process.env.NEXT_PUBLIC_API_URL;
    }
    if (!globalThis.location?.hostname) {
        return null;
    }
    return `http://${globalThis.location.hostname}:3001`;
}

async function resolveGoApiBase(): Promise<string> {
    if (goApiBase !== null && goApiBase !== '') {
        return goApiBase;
    }

    if (isTauri()) {
        const port = await initTauriApiPort();
        if (port !== null && port !== 0) {
            goApiBase = `http://localhost:${port}`;
            return goApiBase;
        }
    }

    const tauriBase = getTauriApiBase();
    if (tauriBase !== null && tauriBase !== '') {
        goApiBase = tauriBase;
        return tauriBase;
    }

    const browserBase = browserApiBase();
    if (browserBase !== null && browserBase !== '') {
        return browserBase;
    }
    return 'http://localhost:3001';
}

export function setApiBase(base: string) {
    goApiBase = base;
}

// hazard: resolved at module load so a synchronous caller (goApiUrl) already reads the Tauri port; a first fetch fired before the webview reports it would otherwise target the browser default
if (globalThis.location !== undefined || isTauri()) {
    void resolveGoApiBase();
}

export type ConnectionStatus = 'connected' | 'disconnected';

let connectionStatus: ConnectionStatus = 'disconnected';
const statusListeners = new Set<(status: ConnectionStatus) => void>();

function setConnectionStatus(status: ConnectionStatus) {
    if (connectionStatus !== status) {
        connectionStatus = status;
        for (const listener of statusListeners) {
            listener(status);
        }
    }
}

export function getConnectionStatus(): ConnectionStatus {
    return connectionStatus;
}

export function onConnectionStatusChange(fn: (status: ConnectionStatus) => void): () => void {
    statusListeners.add(fn);
    return () => {
        statusListeners.delete(fn);
    };
}

// oxlint-disable-next-line typescript/no-unnecessary-type-parameters -- the caller names the shape its route returns; the parameter exists to carry it
function payload<T>(data: unknown): T {
    // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the Go API is this app's own backend; the route's shape is the contract its handler and this caller are both written to
    return data as T;
}

function errorOf(body: string, status: number): string {
    try {
        const error = textOf(JSON.parse(body), 'error');
        return error === '' ? `API error ${status}` : error;
    } catch {
        return `API error ${status}`;
    }
}

class ApiError extends Error {
    constructor(message: string) {
        super(message);
        this.name = 'ApiError';
    }
}

async function apiFetch(path: string, init?: RequestInit): Promise<unknown> {
    try {
        const base = await resolveGoApiBase();
        const res = await fetch(`${base}${path}`, {
            ...init,
            signal: init?.signal ?? AbortSignal.timeout(10_000),
        });
        if (!res.ok) {
            setConnectionStatus('disconnected');
            const body = await res.text().catch(() => '');
            throw new ApiError(errorOf(body, res.status));
        }
        setConnectionStatus('connected');
        const data: unknown = await res.json();
        return data;
    } catch (error) {
        if (error instanceof ApiError) {
            throw error;
        }
        setConnectionStatus('disconnected');
        throw new Error('Failed to connect to API', { cause: error });
    }
}

type RawAgent = {
    name: string;
    role?: string;
    status?: string;
};

// outside: /api/worlds hands two shapes — a world that runs carries its container id, config, creation stamp and agent records; a world the project only declares carries its manifest name and plain-string agents and workspaces (declaredWorldItem in apps/api); `agent` and `workspace` are the older single-valued fields some routes still emit
interface RawWorld extends Omit<
    World,
    'agent' | 'agents' | 'config' | 'created_at' | 'id' | 'status' | 'workspaces'
> {
    agent?: string;
    agents?: (RawAgent | string)[];
    config?: string;
    created_at?: string;
    id?: string;
    status?: World['status'];
    workspace?: string;
    workspaces?: (string | Workspace)[];
}

function normalizeWorldAgent(entry: RawAgent | string, worldStatus: World['status']): Agent {
    if (typeof entry === 'string') {
        return { name: entry, role: 'worker', status: worldStatus };
    }
    return {
        name: entry.name,
        role: entry.role ?? 'worker',
        status: entry.status ?? worldStatus,
    };
}

function normalizeWorkspaces(
    declared: (string | Workspace)[] | undefined,
    legacy: string | undefined,
): undefined | Workspace[] {
    if (declared !== undefined && declared.length > 0) {
        return declared.map((entry) =>
            typeof entry === 'string' ? { name: entry, path: entry } : entry,
        );
    }
    if (legacy === undefined || legacy === '') {
        return undefined;
    }
    return [{ name: 'default', path: legacy }];
}

function normalizeWorlds(data: RawWorld[]): World[] {
    return data.map(({ agent: _agent, workspace: _legacyWs, workspaces, ...w }) => {
        const status = w.status ?? 'idle';
        const agents = w.agents ?? (_agent === undefined ? [] : [_agent]);
        return {
            ...w,
            agent: _agent ?? '',
            agents: agents.map((entry) => normalizeWorldAgent(entry, status)),
            config: w.config ?? w.name ?? '',
            created_at: w.created_at ?? '',
            id: w.id ?? w.name ?? '',
            status,
            workspaces: normalizeWorkspaces(workspaces, _legacyWs),
        };
    });
}

function normalizeAgent(data: Partial<AgentProfile> & { name: string }): AgentProfile {
    return {
        role: data.role ?? 'worker',
        engine: '',
        provider: '',
        purpose: '',
        profile: '',
        traits: [],
        skills: [],
        journal: [],
        ...data,
    };
}

/** Encode a value for use as a URL path segment. Handles spaces, special chars, and non-ASCII - so agent names like "QA Eng" don't break API calls. */
export function encPath(segment: string): string {
    return encodeURIComponent(segment);
}

const AGENT_PATH = /^\/api\/agents\/[^/]+$/u;

export async function apiGet<T>(goPath: string): Promise<T> {
    const data = await apiFetch(goPath);
    if (goPath === '/api/worlds' && Array.isArray(data)) {
        return payload<T>(normalizeWorlds(payload<RawWorld[]>(data)));
    }
    if (AGENT_PATH.test(goPath) && textOf(data, 'name') !== '') {
        return payload<T>(normalizeAgent(payload<Partial<AgentProfile> & { name: string }>(data)));
    }
    return payload<T>(data);
}

export async function apiPost<T>(
    goPath: string,
    body?: unknown,
    options: { signal?: AbortSignal } = {},
): Promise<T> {
    return payload<T>(
        await apiFetch(goPath, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            ...(body === undefined ? {} : { body: JSON.stringify(body) }),
            ...options,
        }),
    );
}

export async function apiPut<T>(goPath: string, body?: unknown): Promise<T> {
    return payload<T>(
        await apiFetch(goPath, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            ...(body === undefined ? {} : { body: JSON.stringify(body) }),
        }),
    );
}

export async function apiDelete(goPath: string): Promise<void> {
    await apiFetch(goPath, { method: 'DELETE' });
}

/** POST that returns { ok, error } shape - used for action buttons. */
export async function apiAction(
    goPath: string,
    body?: unknown,
): Promise<{ ok: boolean; error?: string }> {
    try {
        const base = await resolveGoApiBase();
        const res = await fetch(`${base}${goPath}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            ...(body === undefined ? {} : { body: JSON.stringify(body) }),
            signal: AbortSignal.timeout(10_000),
        });
        const data: unknown = await res.json();
        if (!res.ok) {
            const error = textOf(data, 'error');
            return { ok: false, error: error === '' ? 'Unknown error' : error };
        }
        setConnectionStatus('connected');
        return { ok: true };
    } catch {
        setConnectionStatus('disconnected');
        return { ok: false, error: 'Failed to connect to API' };
    }
}

/** Check if the Go API server is reachable. */
export async function isGoApiAvailable(): Promise<boolean> {
    try {
        const base = await resolveGoApiBase();
        const res = await fetch(`${base}/api/status`, {
            signal: AbortSignal.timeout(2000),
        });
        return res.ok;
    } catch {
        return false;
    }
}

/** Build the full Go API URL. Requires the port to have been resolved already (call resolveGoApiBase() first if unsure). Falls back to localhost:3001 if not yet initialized. */
export function goApiUrl(path: string): string {
    const base = goApiBase ?? getTauriApiBase() ?? browserApiBase() ?? 'http://localhost:3001';
    return `${base}${path}`;
}
