'use client';

import { useEffect, useState } from 'react';

import { apiAction, apiGet } from '../../api/client';
import { textOf } from '../../api/fields';
import { optional } from '../../api/optional';
import type { ArchitectStatus } from '../../domain/architect';
import { progressOf } from './progress';
import type { SpawnProgress } from './progress';

export type ArchitectState = 'offline' | 'running' | 'starting' | 'stopping';

type SetStatus = React.Dispatch<React.SetStateAction<ArchitectStatus | null>>;

interface Control {
    running: boolean;
    setStatus: SetStatus;
    report: (message: string) => void;
}

function withStatus(status: ArchitectStatus['status']) {
    return (previous: ArchitectStatus | null) =>
        previous === null ? previous : { ...previous, status };
}

/** Starts and stops the architect, following a start through the progress the status route reports until it runs or fails. */
export function useArchitectControl({ running, setStatus, report }: Control) {
    const [action, setAction] = useState<'start' | 'stop' | null>(null);
    const [progress, setProgress] = useState<null | SpawnProgress>(null);
    const [following, setFollowing] = useState(false);

    useEffect(() => {
        if (!following) {
            return;
        }
        const poll = async () => {
            const status = await optional(apiGet<unknown>('/api/architect/status'));
            const reported = progressOf(status);
            setProgress(reported);
            if (reported?.error !== undefined && reported.error !== '') {
                setFollowing(false);
                setAction(null);
            } else if (textOf(status, 'status') === 'running') {
                setFollowing(false);
                setAction(null);
                setProgress(null);
                setStatus(withStatus('running'));
                report('Architect started');
            }
        };
        const id = setInterval(() => {
            void poll();
        }, 2000);
        return () => {
            clearInterval(id);
        };
    }, [following, setStatus, report]);

    const start = async () => {
        setAction('start');
        const result = await apiAction('/api/architect/start');
        if (result.ok) {
            setFollowing(true);
        } else {
            report(`Error: ${result.error ?? 'Unknown error'}`);
            setAction(null);
        }
    };
    const stop = async () => {
        setAction('stop');
        const result = await apiAction('/api/architect/stop');
        setAction(null);
        if (result.ok) {
            setStatus(withStatus('stopped'));
        }
        report(result.ok ? 'Architect stopped' : `Error: ${result.error ?? 'Unknown error'}`);
    };

    let state: ArchitectState = running ? 'running' : 'offline';
    if (action === 'start') {
        state = 'starting';
    } else if (action === 'stop') {
        state = 'stopping';
    }
    return { busy: action !== null, progress, start, state, stop };
}
