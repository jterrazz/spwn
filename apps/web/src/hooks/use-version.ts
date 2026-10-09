'use client';

import { useCallback, useState } from 'react';

import { apiGet } from '../api/client';
import { useBackgroundTask } from './use-background-task';

const VERSION_CHECK_INTERVAL = 5 * 60 * 1000;

export interface VersionInfo {
    current: string;
    latest: string;
    updateAvailable: boolean;
    releaseUrl: string;
}

async function latestVersion(): Promise<undefined | VersionInfo> {
    try {
        return await apiGet<VersionInfo>('/api/version');
    } catch {
        return undefined;
    }
}

export function useVersion() {
    const [version, setVersion] = useState<null | VersionInfo>(null);
    const [loading, setLoading] = useState(true);

    const fetchVersion = useCallback(async () => {
        const latest = await latestVersion();
        if (latest !== undefined) {
            setVersion(latest);
        }
        setLoading(false);
    }, []);

    useBackgroundTask(fetchVersion, VERSION_CHECK_INTERVAL);

    return { version, loading };
}
