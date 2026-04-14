/**
 * Tauri integration - detect if running inside the native app
 * and get the dynamic API port.
 */

let cachedPort: null | number = null;
let _initPromise: null | Promise<null | number> = null;

export function isTauri(): boolean {
    return typeof globalThis !== 'undefined' && '__TAURI__' in globalThis;
}

/**
 * Initialize the Tauri API port. Must be called (and awaited) once before
 * any API calls. Safe to call multiple times - subsequent calls return the
 * cached result.
 */
export async function initTauriApiPort(): Promise<null | number> {
    if (!isTauri()) {
        return null;
    }
    if (cachedPort) {
        return cachedPort;
    }
    if (_initPromise) {
        return _initPromise;
    }

    _initPromise = (async () => {
        try {
            const { invoke } = (
                globalThis as unknown as {
                    __TAURI__: { core: { invoke: (cmd: string) => Promise<number> } };
                }
            ).__TAURI__.core;
            cachedPort = await invoke('get_api_port');
            return cachedPort;
        } catch {
            return null;
        }
    })();

    return _initPromise;
}

export function getTauriApiBase(): null | string {
    if (cachedPort) {
        return `http://localhost:${cachedPort}`;
    }
    return null;
}
