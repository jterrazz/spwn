'use client';

import { createContext, useCallback, useContext, useMemo, useState } from 'react';

import { apiGet } from '../api/client';
import { useBackgroundTask } from '../hooks/use-background-task';

export interface DockerStatus {
    installed: boolean;
    running: boolean;
    version?: string;
    error?: string;
    hint?: string;
    platform: string;
}

interface DockerContextValue {
    status: DockerStatus | null | undefined;
    lastChecked: null | number;
    refresh: () => Promise<void>;
    ready: boolean;
}

const DockerContext = createContext<DockerContextValue>({
    status: null,
    lastChecked: null,
    refresh: async () => {
        await Promise.resolve();
    },
    ready: false,
});

const POLL_INTERVAL_MS = 3000;

/** Single source of truth for Docker daemon health. Polls the Go API every 3 seconds and exposes the latest probe result through React context. Status semantics: null       - first load, we don't know yet (render skeleton, not lock) undefined  - the API itself is unreachable (different failure mode) object     - actual probe result; check `installed && running` */
export function DockerProvider({ children }: { children: React.ReactNode }) {
    const [status, setStatus] = useState<DockerStatus | null | undefined>(null);
    const [lastChecked, setLastChecked] = useState<null | number>(null);

    const refresh = useCallback(async () => {
        try {
            setStatus(await apiGet<DockerStatus>('/api/system/docker'));
        } catch {
            setStatus(undefined);
        }
        setLastChecked(Date.now());
    }, []);

    useBackgroundTask(refresh, POLL_INTERVAL_MS);

    const ready = status?.installed === true && status.running;

    const contextValue = useMemo(
        () => ({ status, lastChecked, refresh, ready }),
        [status, lastChecked, refresh, ready],
    );

    return <DockerContext.Provider value={contextValue}>{children}</DockerContext.Provider>;
}

export function useDocker(): DockerContextValue {
    return useContext(DockerContext);
}
