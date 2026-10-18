import { fieldOf, numberOf, textOf } from '../../api/fields';

export interface SpawnProgress {
    event: string;
    detail: string;
    elapsedSeconds: number;
    error: string;
    logTail: string;
}

// outside: keyed by the stable event names Universe.StartArchitectDaemonWithOpts emits; an event missing here falls back to its humanised key
const PROGRESS_LABELS: Record<string, string> = {
    already_running: 'Architect already running',
    cleanup: 'Cleaning up old container...',
    container_creating: 'Creating container...',
    container_starting: 'Starting container...',
    credentials_sync: 'Syncing credentials...',
    docker_check: 'Connecting to Docker...',
    host_files: 'Preparing host files...',
    image_building: 'Building architect image (first run takes minutes)...',
    image_ready: 'Image ready',
    image_resolve: 'Resolving architect image...',
    queued: 'Preparing to start architect...',
    ready: 'Architect is ready',
};

/** The start's progress as the status route reports it, or null when it reports none. */
export function progressOf(status: unknown): null | SpawnProgress {
    const progress = fieldOf(status, 'progress');
    if (typeof progress !== 'object' || progress === null) {
        return null;
    }
    return {
        detail: textOf(progress, 'detail'),
        elapsedSeconds: numberOf(progress, 'elapsedSeconds') ?? 0,
        error: textOf(progress, 'error'),
        event: textOf(progress, 'event'),
        logTail: textOf(progress, 'logTail'),
    };
}

export function describeProgress(progress: null | SpawnProgress): string {
    if (progress === null) {
        return 'Sending start signal...';
    }
    const label = PROGRESS_LABELS[progress.event] ?? progress.event.replaceAll('_', ' ');
    return progress.detail === '' ? label : `${label} - ${progress.detail}`;
}
