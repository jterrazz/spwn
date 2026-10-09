'use client';

import { useCallback, useState } from 'react';

import { apiAction, apiGet, apiPost } from '../../api/client';
import { fieldOf, textOf } from '../../api/fields';
import { optional } from '../../api/optional';
import { useBackgroundTask } from '../../hooks/use-background-task';
import type { ProviderInfo } from './meta';

/** The providers the host has credentials for, and the verbs that check, clear and reconnect them, each reported through `report`. */
export function useProviders(report: (message: string) => void) {
    const [providers, setProviders] = useState<ProviderInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [failure, setFailure] = useState('');
    const [checking, setChecking] = useState<null | string>(null);

    const fetchProviders = useCallback(async () => {
        try {
            const listed = await apiGet<{ providers?: ProviderInfo[] }>('/api/auth/providers');
            setProviders(listed.providers ?? []);
            setFailure('');
        } catch (error) {
            setFailure(error instanceof Error ? error.message : 'Failed to load providers');
        }
        setLoading(false);
    }, []);
    useBackgroundTask(fetchProviders);

    const check = async (name: string) => {
        setChecking(name);
        const checked = await optional(apiPost<unknown>('/api/auth/check', { provider: name }));
        setChecking(null);
        const connected = fieldOf(checked, 'connected') === true;
        const error = textOf(checked, 'error');
        report(
            connected ? `${name} verified` : `Error: ${error === '' ? `${name} failed` : error}`,
        );
        setProviders((previous) =>
            previous.map((entry) =>
                entry.provider === name
                    ? { ...entry, connected, error: connected ? null : error }
                    : entry,
            ),
        );
        if (connected) {
            await fetchProviders();
        }
    };
    const act = async (path: string, name: string, done: string, failed: string) => {
        const result = await apiAction(path, { provider: name });
        report(result.ok ? done : `Error: ${failed}`);
        if (result.ok) {
            await fetchProviders();
        }
    };
    const reset = async (name: string) => {
        await act('/api/auth/reset', name, `${name} cleared`, 'Failed to reset');
    };
    const reconnect = async (name: string) => {
        await act('/api/auth/reconnect', name, `${name} reconnected`, 'Failed to reconnect');
    };

    return { check, checking, failure, loading, providers, reconnect, reset };
}
