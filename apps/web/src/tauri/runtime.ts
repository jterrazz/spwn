interface UpdateDescriptor {
    available: boolean;
    version?: string;
    notes?: string;
    downloadAndInstall?: () => Promise<void>;
}

/** What the Tauri shell injects into the webview: the core command bridge and the plugins this app enables. */
export interface TauriBridge {
    core?: { invoke: (command: string) => Promise<unknown> };
    updater?: { check: () => Promise<null | UpdateDescriptor> };
    dialog?: {
        ask: (
            message: string,
            options?: { title?: string; kind?: 'info' | 'warning' },
        ) => Promise<boolean>;
    };
    process?: { relaunch: () => Promise<void> };
}

declare global {
    var __TAURI__: TauriBridge | undefined;
}

let cachedPort: null | number = null;
let initPromise: null | Promise<null | number> = null;

export function tauriBridge(): TauriBridge | undefined {
    return globalThis.__TAURI__;
}

export function isTauri(): boolean {
    return tauriBridge() !== undefined;
}

async function askForPort(): Promise<null | number> {
    try {
        const port = await tauriBridge()?.core?.invoke('get_api_port');
        cachedPort = typeof port === 'number' ? port : null;
        return cachedPort;
    } catch {
        return null;
    }
}

/** The port the desktop shell serves the API on; asked once, then cached. Null in a browser. */
export async function initTauriApiPort(): Promise<null | number> {
    if (!isTauri()) {
        return null;
    }
    if (cachedPort !== null && cachedPort !== 0) {
        return cachedPort;
    }
    initPromise ??= askForPort();
    return await initPromise;
}

export function getTauriApiBase(): null | string {
    if (cachedPort !== null && cachedPort !== 0) {
        return `http://localhost:${cachedPort}`;
    }
    return null;
}
