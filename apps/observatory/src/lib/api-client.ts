/**
 * API client for the Go API server.
 * The Go API is the sole backend — no fallback to Next.js API routes.
 */

import type { World, AgentProfile } from "./types";
import { getTauriApiBase, isTauri } from "./tauri";

// Dynamic API base — Tauri app uses a random port, browser defaults to 3001
let _goApiBase: string | null = null;

function getGoApiBase(): string {
  if (_goApiBase) return _goApiBase;
  // Check Tauri first
  const tauriBase = getTauriApiBase();
  if (tauriBase) {
    _goApiBase = tauriBase;
    return tauriBase;
  }
  // Default: use the same hostname as the current page (works on LAN)
  // so 192.168.1.137:3000 → 192.168.1.137:3001
  if (typeof window !== "undefined") {
    return process.env.NEXT_PUBLIC_API_URL || `http://${window.location.hostname}:3001`;
  }
  return "http://localhost:3001";
}

// Allow Tauri to set the port after initialization
export function setApiBase(base: string) {
  _goApiBase = base;
}

// ── Connection status tracking ──

export type ConnectionStatus = "connected" | "disconnected";

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

// ── Core fetch helper ──

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  try {
    const res = await fetch(`${getGoApiBase()}${path}`, {
      ...init,
      signal: init?.signal ?? AbortSignal.timeout(10000),
    });
    if (!res.ok) {
      setConnectionStatus("disconnected");
      const body = await res.text().catch(() => "");
      let msg = `API error ${res.status}`;
      try { const j = JSON.parse(body); if (j.error) msg = j.error; } catch {}
      throw new Error(msg);
    }
    setConnectionStatus("connected");
    return res.json();
  } catch (err) {
    if (err instanceof Error && err.message.startsWith("API error")) {
      throw err;
    }
    setConnectionStatus("disconnected");
    throw new Error("Failed to connect to API");
  }
}

// ── Data normalization ──

/** Raw world data from the Go API (may have `agent` string instead of `agents` array). */
interface RawWorld extends Omit<World, "agent" | "agents" | "status" | "workspaces"> {
  agent?: string;
  agents?: World["agents"];
  status?: World["status"];
  workspaces?: World["workspaces"];
  workspace?: string; // legacy single-workspace field
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
      wsList = [{ name: "default", path: _legacyWs }];
    }
    return {
      ...w,
      agent: _agent ?? "",
      status: w.status || "idle",
      agents: w.agents ?? (_agent ? [{ name: _agent, role: "citizen", status: w.status || "idle" }] : []),
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
    role: 'citizen',
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

/**
 * Encode a value for use as a URL path segment. Handles spaces, special
 * chars, and non-ASCII — so agent names like "QA Eng" don't break API calls.
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
  if (goPath === "/api/universes" && Array.isArray(data)) {
    return normalizeWorlds(data) as T;
  }
  // Normalize agent profile data from Go API
  if (goPath.match(/^\/api\/agents\/[^/]+$/) && data && typeof data === 'object' && 'name' in (data as object)) {
    return normalizeAgent(data as unknown as Partial<AgentProfile> & { name: string }) as T;
  }
  return data;
}

/**
 * POST to Go API.
 */
export async function apiPost<T>(
  goPath: string,
  body?: unknown,
): Promise<T> {
  return apiFetch<T>(goPath, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
}

/**
 * PUT to Go API.
 */
export async function apiPut<T>(
  goPath: string,
  body?: unknown,
): Promise<T> {
  return apiFetch<T>(goPath, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
}

/**
 * DELETE on Go API.
 */
export async function apiDelete(goPath: string): Promise<void> {
  await apiFetch<unknown>(goPath, { method: "DELETE" });
}

/**
 * POST that returns { ok, error } shape — used for action buttons.
 */
export async function apiAction(
  goPath: string,
  body?: unknown,
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`${getGoApiBase()}${goPath}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: body ? JSON.stringify(body) : undefined,
      signal: AbortSignal.timeout(10000),
    });
    const data = await res.json();
    if (!res.ok) return { ok: false, error: data.error || "Unknown error" };
    setConnectionStatus("connected");
    return { ok: true };
  } catch {
    setConnectionStatus("disconnected");
    return { ok: false, error: "Failed to connect to API" };
  }
}

/**
 * Check if the Go API server is reachable.
 */
export async function isGoApiAvailable(): Promise<boolean> {
  try {
    const res = await fetch(`${getGoApiBase()}/api/status`, {
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
  return `${getGoApiBase()}${path}`;
}
