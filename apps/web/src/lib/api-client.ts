/**
 * API client for the Go API server.
 * The Go API is the sole backend - no fallback to Next.js API routes.
 */

import { getTauriApiBase, initTauriApiPort, isTauri } from './tauri';
import type { AgentProfile, World } from './types';

// Dynamic API base - Tauri app uses a random port, browser defaults to 3001
let _goApiBase: null | string = null;

/**
 * Resolves the Go API base URL. In Tauri mode, waits for the port to be
 * retrieved from the Rust backend before returning. This ensures every API
 * call targets the correct port - even if the very first fetch fires before
 * the Tauri webview has finished initializing.
 */
async function resolveGoApiBase(): Promise<string> {
    if (_goApiBase) {
        return _goApiBase;
    }

    // In Tauri, wait for the port from the Rust side
    if (isTauri()) {
        const port = await initTauriApiPort();
        if (port) {
            _goApiBase = `http://localhost:${port}`;
            return _goApiBase;
        }
    }

    // Check if the port was already cached synchronously
    const tauriBase = getTauriApiBase();
    if (tauriBase) {
        _goApiBase = tauriBase;
        return tauriBase;
    }

    // Browser fallback
    if (typeof globalThis !== 'undefined') {
        return process.env.NEXT_PUBLIC_API_URL || `http://${globalThis.location.hostname}:3001`;
    }
    return 'http://localhost:3001';
}

// Allow external code to set the port explicitly
export function setApiBase(base: string) {
    _goApiBase = base;
}

// Eagerly resolve the API base on module load so that even synchronous
// Callers (goApiUrl) get the right value as soon as possible. The promise
// Is fire-and-forget - by the time a user interaction triggers a fetch,
// The port will be cached.
if (typeof globalThis !== 'undefined') {
    void resolveGoApiBase();
}

// ── Connection status tracking ──

export type ConnectionStatus = 'connected' | 'disconnected';

let _connectionStatus: ConnectionStatus = 'disconnected';
const _statusListeners = new Set<(status: ConnectionStatus) => void>();

function setConnectionStatus(status: ConnectionStatus) {
    if (_connectionStatus !== status) {
        _connectionStatus = status;
        _statusListeners.forEach((fn) => fn(status));
    }
}

export function getConnectionStatus(): ConnectionStatus {
    return _connectionStatus;
}

export function onConnectionStatusChange(fn: (status: ConnectionStatus) => void): () => void {
    _statusListeners.add(fn);
    return () => {
        _statusListeners.delete(fn);
    };
}

// ── Core fetch helper ──

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
    try {
        const base = await resolveGoApiBase();
        const res = await fetch(`${base}${path}`, {
            ...init,
            signal: init?.signal ?? AbortSignal.timeout(10_000),
        });
        if (!res.ok) {
            setConnectionStatus('disconnected');
            const body = await res.text().catch(() => '');
            let msg = `API error ${res.status}`;
            try {
                const j = JSON.parse(body);
                if (j.error) {
                    msg = j.error;
                }
            } catch {}
            throw new Error(msg);
        }
        setConnectionStatus('connected');
        return res.json();
    } catch (error) {
        if (error instanceof Error && error.message.startsWith('API error')) {
            throw error;
        }
        setConnectionStatus('disconnected');
        throw new Error('Failed to connect to API', { cause: error });
    }
}

// ── Data normalization ──

/** Raw world data from the Go API (may have `agent` string instead of `agents` array). */
interface RawWorld extends Omit<World, 'agent' | 'agents' | 'status' | 'workspaces'> {
    agent?: string;
    agents?: World['agents'];
    status?: World['status'];
    workspaces?: World['workspaces'];
    workspace?: string; // Legacy single-workspace field
}

/**
 * Normalize Go API world data to match frontend World interface.
 * Go returns `agent` (string), frontend expects `agents` (array).
 * Also migrates legacy `workspace` string into `workspaces` array.
 */
function normalizeWorlds(data: RawWorld[]): World[] {
    return data.map(({ agent: _agent, workspace: _legacyWs, workspaces, ...w }) => {
        let wsList = workspaces;
        if ((!wsList || wsList.length === 0) && _legacyWs) {
            wsList = [{ name: 'default', path: _legacyWs }];
        }
        return {
            ...w,
            agent: _agent ?? '',
            status: w.status || 'idle',
            agents: (
                w.agents ??
                (_agent ? [{ name: _agent, role: 'worker', status: w.status || 'idle' }] : [])
            ).map(
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                (a: any) => ({ ...a, role: a.role || 'worker' }),
            ),
            workspaces: wsList,
        };
    });
}

/**
 * Normalize Go API agent data to match frontend AgentProfile interface.
 * Fills in defaults for any missing fields so the UI never crashes on undefined arrays.
 */
function normalizeAgent(data: Partial<AgentProfile> & { name: string }): AgentProfile {
    return {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        role: (data as any).role || 'worker',
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

/**
 * Encode a value for use as a URL path segment. Handles spaces, special
 * chars, and non-ASCII - so agent names like "QA Eng" don't break API calls.
 */
export function encPath(segment: string): string {
    return encodeURIComponent(segment);
}

// ── Public API ──

/**
 * GET from Go API.
 */
export async function apiGet<T>(goPath: string): Promise<T> {
    const data = await apiFetch<T>(goPath);
    // Normalize world data from Go API
    if (goPath === '/api/worlds' && Array.isArray(data)) {
        return normalizeWorlds(data) as T;
    }
    // Normalize agent profile data from Go API
    if (
        goPath.match(/^\/api\/agents\/[^/]+$/) &&
        data &&
        typeof data === 'object' &&
        'name' in (data as object)
    ) {
        return normalizeAgent(data as unknown as Partial<AgentProfile> & { name: string }) as T;
    }
    return data;
}

/**
 * POST to Go API.
 */
export async function apiPost<T>(goPath: string, body?: unknown): Promise<T> {
    return apiFetch<T>(goPath, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : undefined,
    });
}

/**
 * PUT to Go API.
 */
export async function apiPut<T>(goPath: string, body?: unknown): Promise<T> {
    return apiFetch<T>(goPath, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : undefined,
    });
}

/**
 * DELETE on Go API.
 */
export async function apiDelete(goPath: string): Promise<void> {
    await apiFetch<unknown>(goPath, { method: 'DELETE' });
}

/**
 * POST that returns { ok, error } shape - used for action buttons.
 */
export async function apiAction(
    goPath: string,
    body?: unknown,
): Promise<{ ok: boolean; error?: string }> {
    try {
        const base = await resolveGoApiBase();
        const res = await fetch(`${base}${goPath}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: body ? JSON.stringify(body) : undefined,
            signal: AbortSignal.timeout(10_000),
        });
        const data = await res.json();
        if (!res.ok) {
            return { ok: false, error: data.error || 'Unknown error' };
        }
        setConnectionStatus('connected');
        return { ok: true };
    } catch {
        setConnectionStatus('disconnected');
        return { ok: false, error: 'Failed to connect to API' };
    }
}

/**
 * Check if the Go API server is reachable.
 */
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

/**
 * Build the full Go API URL for a path (useful for direct fetch calls).
 */
/**
 * Build the full Go API URL. Requires the port to have been resolved
 * already (call resolveGoApiBase() first if unsure). Falls back to
 * localhost:3001 if not yet initialized.
 */
export function goApiUrl(path: string): string {
    const base =
        _goApiBase ||
        getTauriApiBase() ||
        (typeof globalThis !== 'undefined'
            ? process.env.NEXT_PUBLIC_API_URL || `http://${globalThis.location.hostname}:3001`
            : 'http://localhost:3001');
    return `${base}${path}`;
}
