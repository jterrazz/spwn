'use client';

import {
    IconArrowRight,
    IconBrandDocker,
    IconCheck,
    IconExternalLink,
    IconRefresh,
} from '@tabler/icons-react';
import Link from 'next/link';

import type { OnboardingStatus } from './steps';

export const STEP_BUTTON =
    'text-foreground/90 inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-[11px] font-medium hover:bg-white/[0.08]';

export function Continue({ onAdvance }: { onAdvance: () => void }) {
    return (
        <button className={STEP_BUTTON} onClick={onAdvance} type="button">
            Continue <IconArrowRight size={11} />
        </button>
    );
}

function dockerLine(status: OnboardingStatus['docker'] | undefined): string {
    if (status?.installed === true && status.running) {
        return `Docker ${status.version === undefined || status.version === '' ? '' : `v${status.version}`} running`;
    }
    return status?.installed === true
        ? 'Docker is installed but the daemon is not running'
        : 'Docker is not installed on this machine';
}

function DockerActions({
    docker,
    refreshing,
    onRefresh,
}: {
    docker: OnboardingStatus['docker'] | undefined;
    refreshing: boolean;
    onRefresh: () => void;
}) {
    return (
        <div className="flex flex-wrap items-center gap-2">
            {docker?.installed !== true && (
                <a
                    className={STEP_BUTTON}
                    href={
                        docker?.platform === 'linux'
                            ? 'https://docs.docker.com/engine/install/'
                            : 'https://www.docker.com/products/docker-desktop/'
                    }
                    rel="noreferrer"
                    target="_blank"
                >
                    Install Docker
                    <IconExternalLink size={11} />
                </a>
            )}
            <button
                className={`${STEP_BUTTON} disabled:opacity-50`}
                disabled={refreshing}
                onClick={onRefresh}
                type="button"
            >
                <IconRefresh className={refreshing ? 'animate-spin' : ''} size={11} />
                Recheck
            </button>
        </div>
    );
}

interface DockerStepProps {
    status: null | OnboardingStatus;
    refreshing: boolean;
    onRefresh: () => void;
    onAdvance: () => void;
}

export function DockerStep({ status, refreshing, onRefresh, onAdvance }: DockerStepProps) {
    const docker = status?.docker;
    const ok = docker?.installed === true && docker.running;
    const hint = docker?.hint ?? '';
    return (
        <div className="space-y-3">
            <div
                className={`rounded-lg border px-3 py-2.5 text-xs ${ok ? 'border-emerald-400/30 bg-emerald-500/[0.06] text-emerald-100' : 'border-red-400/30 bg-red-500/[0.06] text-red-100'}`}
            >
                <div className="flex items-center gap-2">
                    <IconBrandDocker size={14} />
                    <span className="font-medium">{dockerLine(docker)}</span>
                </div>
                {!ok && hint !== '' && (
                    <p className="mt-1 ml-[22px] text-[11px] opacity-80">{hint}</p>
                )}
            </div>
            {ok ? (
                <Continue onAdvance={onAdvance} />
            ) : (
                <DockerActions docker={docker} onRefresh={onRefresh} refreshing={refreshing} />
            )}
        </div>
    );
}

export function WorldStep({
    status,
    onAdvance,
}: {
    status: null | OnboardingStatus;
    onAdvance: () => void;
}) {
    const detected = status?.hasWorlds === true;
    return (
        <div className="space-y-3">
            <p className="text-muted-foreground/60 text-[11px]">
                Open the worlds page and click{' '}
                <span className="text-foreground/80">Spawn world</span>. We&apos;ll detect it as
                soon as a world is created.
            </p>
            <div className="flex items-center gap-2">
                <Link className={STEP_BUTTON} href="/">
                    Open worlds
                    <IconArrowRight size={11} />
                </Link>
                {detected && (
                    <span className="inline-flex items-center gap-1 text-[11px] text-emerald-300/80">
                        <IconCheck size={11} />
                        World detected
                    </span>
                )}
            </div>
            {detected && <Continue onAdvance={onAdvance} />}
        </div>
    );
}

export function ChatStep() {
    return (
        <div className="space-y-3">
            <p className="text-muted-foreground/60 text-[11px]">
                Open any world from the sidebar and send a message in the chat panel. Once you
                finish here, click <span className="text-foreground/80">Finish setup</span> below.
            </p>
            <Link className={STEP_BUTTON} href="/">
                Go to worlds <IconArrowRight size={11} />
            </Link>
        </div>
    );
}
