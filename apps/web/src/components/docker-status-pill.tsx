'use client';

import { IconBrandDocker } from '@tabler/icons-react';

import { useDocker } from '@/contexts/docker-context';

/**
 * Compact Docker status indicator for the sidebar footer. Always visible
 * so the user can see daemon health at a glance from any screen. Clicking
 * forces a refresh.
 */
export function DockerStatusPill() {
    const { status, ready, refresh } = useDocker();

    // First load: render a neutral placeholder so we don't flash red.
    const isLoading = status === null;
    const apiDown = status === undefined;

    let label: string;
    let dotClass: string;
    let title: string;

    if (isLoading) {
        label = 'checking';
        dotClass = 'bg-muted-foreground/40';
        title = 'Checking Docker daemon…';
    } else if (apiDown) {
        label = 'API offline';
        dotClass = 'bg-amber-400';
        title = 'Cannot reach the spwn API';
    } else if (ready) {
        label = status?.version ? `v${status.version}` : 'connected';
        dotClass = 'bg-emerald-400';
        title = `Docker daemon running${status?.version ? ` (v${status.version})` : ''}`;
    } else {
        label = 'offline';
        dotClass = 'bg-red-400';
        title = status?.error || 'Docker daemon is not reachable';
    }

    return (
        <button
            aria-label={`Docker status: ${label}`}
            className="group inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 text-[10px] uppercase tracking-wider text-muted-foreground/50 transition-colors hover:bg-white/[0.04] hover:text-foreground/80"
            onClick={() => void refresh()}
            title={title}
            type="button"
        >
            <IconBrandDocker className="opacity-60" size={12} />
            <span className="relative flex h-1.5 w-1.5">
                {ready && (
                    <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400/60 opacity-75" />
                )}
                <span className={`relative inline-flex h-1.5 w-1.5 rounded-full ${dotClass}`} />
            </span>
            <span className="font-mono">{label}</span>
        </button>
    );
}
