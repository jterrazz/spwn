'use client';

import { useCallback, useEffect, useState } from 'react';

import { getConnectionStatus, isGoApiAvailable, onConnectionStatusChange } from '../../api/client';
import type { ConnectionStatus } from '../../api/client';
import { useBackgroundTask } from '../../hooks/use-background-task';

/** Whether the API answers: every request reports it, and a probe asks every ten seconds. */
export function useConnection(): ConnectionStatus {
    const [connection, setConnection] = useState<ConnectionStatus>(getConnectionStatus());
    useEffect(() => onConnectionStatusChange(setConnection), []);
    const probe = useCallback(async () => {
        setConnection((await isGoApiAvailable()) ? 'connected' : 'disconnected');
    }, []);
    useBackgroundTask(probe, 10_000);
    return connection;
}
