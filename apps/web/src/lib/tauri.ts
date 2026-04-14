/**
 * Tauri integration - detect if running inside the native app
 * and get the dynamic API port.
 */

let cachedPort: number | null = null;
let _initPromise: Promise<number | null> | null = null;

export function isTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI__" in window;
}

/**
 * Initialize the Tauri API port. Must be called (and awaited) once before
 * any API calls. Safe to call multiple times - subsequent calls return the
 * cached result.
 */
export async function initTauriApiPort(): Promise<number | null> {
  if (!isTauri()) return null;
  if (cachedPort) return cachedPort;
  if (_initPromise) return _initPromise;

  _initPromise = (async () => {
    try {
      // @ts-expect-error Tauri globals
      const { invoke } = window.__TAURI__.core;
      cachedPort = await invoke("get_api_port");
      return cachedPort;
    } catch {
      return null;
    }
  })();

  return _initPromise;
}

export function getTauriApiBase(): string | null {
  if (cachedPort) return `http://localhost:${cachedPort}`;
  return null;
}
