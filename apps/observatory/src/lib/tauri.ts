/**
 * Tauri integration — detect if running inside the native app
 * and get the dynamic API port.
 */

let cachedPort: number | null = null;

export function isTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI__" in window;
}

export async function getTauriApiPort(): Promise<number | null> {
  if (!isTauri()) return null;
  if (cachedPort) return cachedPort;

  try {
    // @ts-expect-error Tauri globals
    const { invoke } = window.__TAURI__.core;
    cachedPort = await invoke("get_api_port");
    return cachedPort;
  } catch {
    return null;
  }
}

export function getTauriApiBase(): string | null {
  if (cachedPort) return `http://localhost:${cachedPort}`;
  return null;
}
