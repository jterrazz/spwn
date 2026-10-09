'use client';

import { IconBrandDocker } from '@tabler/icons-react';

import { useDocker } from '../contexts/docker-context';
import type { DockerStatus } from '../contexts/docker-context';
import { inBackground } from '../events/in-background';

interface Reading {
    label: string;
    dot: string;
    title: string;
}

function readingOf(status: DockerStatus | null | undefined, ready: boolean): Reading {
    if (status === null) {
        return {
            dot: 'bg-muted-foreground/40',
            label: 'checking',
            title: 'Checking Docker daemon…',
        };
    }
    if (status === undefined) {
        return { dot: 'bg-amber-400', label: 'API offline', title: 'Cannot reach the spwn API' };
    }
    if (!ready) {
        return {
            dot: 'bg-red-400',
            label: 'offline',
            title: status.error ?? 'Docker daemon is not reachable',
        };
    }
    const version = status.version ?? '';
    return {
        dot: 'bg-emerald-400',
        label: version === '' ? 'connected' : `v${version}`,
        title: `Docker daemon running${version === '' ? '' : ` (v${version})`}`,
    };
}

/** Compact Docker status indicator for the sidebar footer. Always visible so the user can see daemon health at a glance from any screen. Clicking forces a refresh. */
export function DockerStatusPill() {
    const { status, ready, refresh } = useDocker();
    const { dot, label, title } = readingOf(status, ready);

    return (
        <button
            aria-label={`Docker status: ${label}`}
            className="group text-muted-foreground/50 hover:text-foreground/80 inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 text-[10px] tracking-wider uppercase transition-colors hover:bg-white/[0.04]"
            onClick={inBackground(refresh)}
            title={title}
            type="button"
        >
            <IconBrandDocker className="opacity-60" size={12} />
            <span className="relative flex h-1.5 w-1.5">
                {ready && (
                    <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400/60 opacity-75" />
                )}
                <span className={`relative inline-flex h-1.5 w-1.5 rounded-full ${dot}`} />
            </span>
            <span className="font-mono">{label}</span>
        </button>
    );
}
