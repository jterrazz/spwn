'use client';

import { useCallback, useState } from 'react';

import { apiGet } from '../api/client';
import { useBackgroundTask } from './use-background-task';

/** A list the API serves at `path`, loaded once mounted; empty until it arrives, and when it cannot. */
export function useList<Item>(path: string): Item[] {
    const [items, setItems] = useState<Item[]>([]);
    const load = useCallback(async () => {
        try {
            setItems(await apiGet<Item[]>(path));
        } catch {
            setItems([]);
        }
    }, [path]);
    useBackgroundTask(load);
    return items;
}
