'use client';

import { IconBrandDocker, IconExternalLink, IconLoader2, IconRefresh } from '@tabler/icons-react';
import { useState } from 'react';

import { useDocker } from '../contexts/docker-context';
import type { DockerStatus } from '../contexts/docker-context';
import { inBackground } from '../events/in-background';
import { useClock } from '../hooks/use-clock';

interface Lock {
    title: string;
    subtitle: string;
    installUrl: string | undefined;
    error: string | undefined;
    hint: string | undefined;
}

function lockOf(status: DockerStatus | null | undefined): Lock | undefined {
    if (status === undefined) {
        return {
            error: undefined,
            hint: undefined,
            installUrl: undefined,
            subtitle: 'Start Docker Desktop and spwn will pick it up automatically.',
            title: 'Waiting for Docker',
        };
    }
    if (status === null || (status.installed && status.running)) {
        return undefined;
    }
    const missing = !status.installed;
    return {
        error: status.error,
        hint: status.hint,
        installUrl: missing
            ? status.platform === 'linux'
                ? 'https://docs.docker.com/engine/install/'
                : 'https://www.docker.com/products/docker-desktop/'
            : undefined,
        subtitle: missing
            ? 'Every spwn world runs inside a Docker container. Install Docker to continue.'
            : 'Start Docker Desktop and spwn will pick it up automatically.',
        title: missing ? "Docker isn't installed" : 'Waiting for Docker',
    };
}

function InstallLink({ href }: { href: string }) {
    return (
        <a
            className="text-foreground/90 inline-flex items-center gap-1.5 rounded-lg border border-white/15 bg-white/[0.06] px-4 py-2 text-xs font-medium transition-colors hover:bg-white/[0.1]"
            href={href}
            rel="noreferrer"
            target="_blank"
        >
            <IconBrandDocker size={14} />
            Install Docker
            <IconExternalLink size={11} />
        </a>
    );
}

/** Full-area lock screen shown whenever the host Docker daemon is not reachable. Replaces page content (the sidebar still renders, dimmed) so users cannot trigger any action that depends on Docker. The component is intentionally calm: spwn IS a Docker control plane, so "Docker offline" is the system state, not an emergency. We explain it, keep retrying in the background, and recover automatically. */
export function DockerLockScreen() {
    const { status, lastChecked, refresh } = useDocker();
    const [refreshing, setRefreshing] = useState(false);
    const now = useClock();
    const lock = lockOf(status);

    if (lock === undefined) {
        return null;
    }
    const retry = async () => {
        setRefreshing(true);
        await refresh();
        setRefreshing(false);
    };
    const secondsAgo = lastChecked === null ? 0 : Math.max(0, now - Math.floor(lastChecked / 1000));

    return (
        <LockShell
            error={lock.error}
            hint={lock.hint}
            icon={<DockerPulse />}
            primaryAction={
                <div className="flex flex-wrap items-center justify-center gap-2">
                    {lock.installUrl !== undefined && <InstallLink href={lock.installUrl} />}
                    <RetryButton loading={refreshing} onClick={inBackground(retry)} />
                </div>
            }
            secondsAgo={secondsAgo}
            subtitle={lock.subtitle}
            title={lock.title}
        />
    );
}

function LockDetails({ error, hint }: { error: string; hint: string }) {
    if (error === '' && hint === '') {
        return null;
    }
    return (
        <div className="mt-5 space-y-1.5 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5 text-left">
            {error !== '' && (
                <p className="text-muted-foreground/70 font-mono text-[10.5px] leading-snug break-words">
                    {error}
                </p>
            )}
            {hint !== '' && <p className="text-foreground/70 text-[11px]">{hint}</p>}
        </div>
    );
}

function LockShell({
    icon,
    title,
    subtitle,
    hint,
    error,
    primaryAction,
    secondsAgo,
}: {
    icon: React.ReactNode;
    title: string;
    subtitle: string;
    hint?: string | undefined;
    error?: string | undefined;
    primaryAction: React.ReactNode;
    secondsAgo: number;
}) {
    return (
        <div className="relative flex h-full w-full items-center justify-center px-6 py-10">
            <div
                aria-hidden
                className="bg-gradient-radial pointer-events-none absolute inset-0 from-red-500/20 via-transparent to-transparent"
            />

            <div className="relative z-10 w-full max-w-md">
                <div className="rounded-2xl border border-white/[0.08] bg-black/30 px-8 py-9 text-center shadow-2xl backdrop-blur-md">
                    <div className="mb-5 flex justify-center">{icon}</div>
                    <h1 className="font-heading text-foreground/95 text-xl tracking-wide">
                        {title}
                    </h1>
                    <p className="text-muted-foreground/70 mx-auto mt-2 max-w-sm text-[12px] leading-relaxed">
                        {subtitle}
                    </p>

                    <LockDetails error={error ?? ''} hint={hint ?? ''} />

                    <div className="mt-6">{primaryAction}</div>

                    <div className="text-muted-foreground/40 mt-5 flex items-center justify-center gap-1.5 text-[10px] tracking-wider uppercase">
                        <span className="relative flex h-1.5 w-1.5">
                            <span className="bg-muted-foreground/40 absolute inline-flex h-full w-full animate-ping rounded-full opacity-75" />
                            <span className="bg-muted-foreground/60 relative inline-flex h-1.5 w-1.5 rounded-full" />
                        </span>
                        Checking every 3s · last check {secondsAgo}s ago
                    </div>
                </div>
            </div>
        </div>
    );
}

function DockerPulse() {
    return (
        <div className="relative flex h-16 w-16 items-center justify-center">
            <span
                className="absolute inset-0 rounded-full border border-red-400/30"
                style={{ animation: 'docker-pulse 2.4s ease-out infinite' }}
            />
            <span
                className="absolute inset-2 rounded-full border border-red-400/20"
                style={{ animation: 'docker-pulse 2.4s ease-out infinite 0.6s' }}
            />
            <IconBrandDocker className="relative text-red-300/90" size={32} />
            <style jsx>{`
                @keyframes docker-pulse {
                    0% {
                        transform: scale(0.85);
                        opacity: 0.9;
                    }
                    100% {
                        transform: scale(1.6);
                        opacity: 0;
                    }
                }
            `}</style>
        </div>
    );
}

function RetryButton({ onClick, loading }: { onClick: () => void; loading: boolean }) {
    return (
        <button
            className="text-foreground/90 inline-flex items-center gap-1.5 rounded-lg border border-white/15 bg-white/[0.06] px-4 py-2 text-xs font-medium transition-colors hover:bg-white/[0.1] disabled:opacity-60"
            disabled={loading}
            onClick={onClick}
            type="button"
        >
            {loading ? (
                <IconLoader2 className="animate-spin" size={13} />
            ) : (
                <IconRefresh size={13} />
            )}
            Retry now
        </button>
    );
}
