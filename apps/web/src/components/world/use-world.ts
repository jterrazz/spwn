'use client';

import { useCallback, useState } from 'react';

import { apiGet } from '../../api/client';
import { listOf, textOf } from '../../api/fields';
import type { World } from '../../domain/model';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { closedLog, logsPanelOf } from './logs';
import type { LogLine, SettledLog } from './logs';

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

async function activityOf(worldId: string): Promise<LogLine[]> {
    try {
        const activity = await apiGet<unknown>(`/api/activity?world=${worldId}&limit=200`);
        return listOf(activity, 'events').map(lineOf);
    } catch {
        return [];
    }
}

/** The world's activity log, loaded each time the logs panel opens; what the panel shows of it is `logsPanelOf`'s. */
export function useWorldLogs(worldId: string, open: boolean) {
    const [settled, setSettled] = useState<null | SettledLog>(null);

    const load = useCallback(async () => {
        if (!open) {
            setSettled(closedLog);
            return;
        }
        setSettled({ lines: await activityOf(worldId), thisOpening: true, worldId });
    }, [open, worldId]);
    useBackgroundTask(load);

    return logsPanelOf(settled, worldId, open);
}
