/**
 * API client for the Go API server (port 3001).
 * Falls back to Next.js API routes if the Go API is unavailable.
 *
 * Client components call these functions directly — they try the Go API first
 * (fast, native) and fall back to the local /api/* Next.js routes (exec-based).
 */

const GO_API_BASE =
  typeof window !== "undefined"
    ? (process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001")
    : "http://localhost:3001";

// ── Core fetch helpers ──

async function tryGoApi<T>(path: string, init?: RequestInit): Promise<T | null> {
  try {
    const res = await fetch(`${GO_API_BASE}${path}`, {
      ...init,
      signal: AbortSignal.timeout(3000),
    });
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

async function fallbackNextApi<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

// ── Data normalization ──

/**
 * Normalize Go API world data to match frontend World interface.
 * Go returns `agent` (string), frontend expects `agents` (array).
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function normalizeWorlds(data: any[]): any[] {
  return data.map((w) => ({
    ...w,
    status: w.status || "idle",
    agents: w.agents ?? (w.agent ? [{ name: w.agent, tier: "citizen", status: w.status || "idle" }] : []),
  }));
}

/**
 * Normalize Go API agent data to match frontend AgentProfile interface.
 * Fills in defaults for any missing fields so the UI never crashes on undefined arrays.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function normalizeAgent(data: any): any {
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
      return normalizeAgent(data) as T;
    }
    return data;
  }
  return fallbackNextApi<T>(nextFallback ?? goPath);
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
  return fallbackNextApi<T>(nextFallback ?? goPath, init);
}

/**
 * DELETE on Go API, fall back to Next.js route.
 */
export async function apiDelete(goPath: string, nextFallback?: string): Promise<void> {
  const init: RequestInit = { method: "DELETE" };
  const goRes = await tryGoApi<unknown>(goPath, init);
  if (goRes !== null) return;
  await fallbackNextApi<unknown>(nextFallback ?? goPath, init);
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
    return { ok: true };
  } catch {
    // Fall back to Next.js route
    try {
      const res = await fetch(nextFallback ?? goPath, init);
      const data = await res.json();
      if (!res.ok) return { ok: false, error: data.error || "Unknown error" };
      return { ok: true };
    } catch {
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
