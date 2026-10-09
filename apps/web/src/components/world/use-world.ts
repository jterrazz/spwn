'use client';

import { useCallback, useState } from 'react';

import { apiGet } from '../../api/client';
import { listOf, textOf } from '../../api/fields';
import type { World } from '../../domain/model';
import { useBackgroundTask } from '../../hooks/use-background-task';

export interface LogLine {
    timestamp: string;
    source: string;
    message: string;
}

/** The world the page shows, polled every five seconds; null once it is known not to exist. */
export function useWorld(worldId: string) {
    const [world, setWorld] = useState<null | World>(null);
    const [loading, setLoading] = useState(true);

    const fetchWorld = useCallback(async () => {
        try {
            const worlds = await apiGet<World[]>('/api/worlds');
            setWorld(worlds.find((candidate) => candidate.id === worldId) ?? null);
        } catch {
            setWorld(null);
        }
        setLoading(false);
    }, [worldId]);
    useBackgroundTask(fetchWorld, 5000);

    return { fetchWorld, loading, world };
}

function lineOf(event: unknown): LogLine {
    return {
        message: textOf(event, 'phrase') || textOf(event, 'type'),
        source: textOf(event, 'actor') || 'world',
        timestamp: textOf(event, 'timestamp'),
    };
}

/** The world's activity log, loaded while the logs panel is open. */
export function useWorldLogs(worldId: string, open: boolean) {
    const [lines, setLines] = useState<LogLine[]>([]);
    const [loading, setLoading] = useState(false);

    const load = useCallback(async () => {
        if (!open) {
            return;
        }
        setLoading(true);
        try {
            const activity = await apiGet<unknown>(`/api/activity?world=${worldId}&limit=200`);
            setLines(listOf(activity, 'events').map(lineOf));
        } catch {
            setLines([]);
        }
        setLoading(false);
    }, [open, worldId]);
    useBackgroundTask(load);

    return { lines, loading };
}
