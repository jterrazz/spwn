/**
 * API client for the Go API server (port 3001).
 * Falls back to Next.js API routes if the Go API is unavailable.
 *
 * Client components call these functions directly — they try the Go API first
 * (fast, native) and fall back to the local /api/* Next.js routes (exec-based).
 */

import type { World, AgentProfile } from "./types";

const GO_API_BASE =
  typeof window !== "undefined"
    ? (process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001")
    : "http://localhost:3001";

// ── Connection status tracking ──

export type ConnectionStatus = "connected" | "degraded" | "disconnected";

let _connectionStatus: ConnectionStatus = "disconnected";
const _statusListeners: Set<(status: ConnectionStatus) => void> = new Set();

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
  return () => { _statusListeners.delete(fn); };
}

// ── Core fetch helpers ──

async function tryGoApi<T>(path: string, init?: RequestInit): Promise<T | null> {
  try {
    const res = await fetch(`${GO_API_BASE}${path}`, {
      ...init,
      signal: AbortSignal.timeout(3000),
    });
    if (!res.ok) return null;
    setConnectionStatus("connected");
    return res.json();
  } catch {
    return null;
  }
}

async function fallbackNextApi<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  if (!res.ok) throw new Error(await res.text());
  setConnectionStatus("degraded");
  return res.json();
}

// ── Data normalization ──

/** Raw world data from the Go API (may have `agent` string instead of `agents` array). */
interface RawWorld extends Omit<World, "agent" | "agents" | "status"> {
  agent?: string;
  agents?: World["agents"];
  status?: World["status"];
}

/**
 * Normalize Go API world data to match frontend World interface.
 * Go returns `agent` (string), frontend expects `agents` (array).
 */
function normalizeWorlds(data: RawWorld[]): World[] {
  return data.map(({ agent: _agent, ...w }) => ({
    ...w,
    agent: _agent ?? "",
    status: w.status || "idle",
    agents: w.agents ?? (_agent ? [{ name: _agent, tier: "citizen", status: w.status || "idle" }] : []),
  }));
}

/**
 * Normalize Go API agent data to match frontend AgentProfile interface.
 * Fills in defaults for any missing fields so the UI never crashes on undefined arrays.
 */
function normalizeAgent(data: Partial<AgentProfile> & { name: string }): AgentProfile {
  return {
    tier: 'citizen',
    engine: 'claude-code',
    provider: 'anthropic',
    purpose: '',
    persona: '',
    traits: [],
    skills: [],
    playbooks: [],
    knowledge: [],
    journal: [],
    bonds: [],
    ...data,
  };
}

// ── Public API ──

/**
 * GET from Go API, fall back to Next.js route.
 */
export async function apiGet<T>(goPath: string, nextFallback?: string): Promise<T> {
  const data = await tryGoApi<T>(goPath);
  if (data !== null) {
    // Normalize world data from Go API
    if (goPath === "/api/universes" && Array.isArray(data)) {
      return normalizeWorlds(data) as T;
    }
    // Normalize agent profile data from Go API
    if (goPath.match(/^\/api\/agents\/[^/]+$/) && data && typeof data === 'object' && 'name' in (data as object)) {
      return normalizeAgent(data as unknown as Partial<AgentProfile> & { name: string }) as T;
    }
    return data;
  }
  try {
    return await fallbackNextApi<T>(nextFallback ?? goPath);
  } catch {
    setConnectionStatus("disconnected");
    throw new Error("Failed to connect to any API");
  }
}

/**
 * POST to Go API, fall back to Next.js route.
 */
export async function apiPost<T>(
  goPath: string,
  body?: unknown,
  nextFallback?: string
): Promise<T> {
  const init: RequestInit = {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  };

  const data = await tryGoApi<T>(goPath, init);
  if (data !== null) return data;
  try {
    return await fallbackNextApi<T>(nextFallback ?? goPath, init);
  } catch {
    setConnectionStatus("disconnected");
    throw new Error("Failed to connect to any API");
  }
}

/**
 * PUT to Go API, fall back to Next.js route.
 */
export async function apiPut<T>(
  goPath: string,
  body?: unknown,
  nextFallback?: string
): Promise<T> {
  const init: RequestInit = {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  };

  const data = await tryGoApi<T>(goPath, init);
  if (data !== null) return data;
  try {
    return await fallbackNextApi<T>(nextFallback ?? goPath, init);
  } catch {
    setConnectionStatus("disconnected");
    throw new Error("Failed to connect to any API");
  }
}

/**
 * DELETE on Go API, fall back to Next.js route.
 */
export async function apiDelete(goPath: string, nextFallback?: string): Promise<void> {
  const init: RequestInit = { method: "DELETE" };
  const goRes = await tryGoApi<unknown>(goPath, init);
  if (goRes !== null) return;
  try {
    await fallbackNextApi<unknown>(nextFallback ?? goPath, init);
  } catch {
    setConnectionStatus("disconnected");
    throw new Error("Failed to connect to any API");
  }
}

/**
 * POST that returns { ok, error } shape — used for action buttons.
 * Tries Go API first, then Next.js fallback.
 */
export async function apiAction(
  goPath: string,
  body?: unknown,
  nextFallback?: string
): Promise<{ ok: boolean; error?: string }> {
  const init: RequestInit = {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  };

  try {
    const res = await fetch(`${GO_API_BASE}${goPath}`, {
      ...init,
      signal: AbortSignal.timeout(10000),
    });
    const data = await res.json();
    if (!res.ok) return { ok: false, error: data.error || "Unknown error" };
    setConnectionStatus("connected");
    return { ok: true };
  } catch {
    // Fall back to Next.js route
    try {
      const res = await fetch(nextFallback ?? goPath, init);
      const data = await res.json();
      if (!res.ok) return { ok: false, error: data.error || "Unknown error" };
      setConnectionStatus("degraded");
      return { ok: true };
    } catch {
      setConnectionStatus("disconnected");
      return { ok: false, error: "Failed to connect to API" };
    }
  }
}

/**
 * Check if the Go API server is reachable.
 */
export async function isGoApiAvailable(): Promise<boolean> {
  try {
    const res = await fetch(`${GO_API_BASE}/api/status`, {
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
export function goApiUrl(path: string): string {
  return `${GO_API_BASE}${path}`;
}
