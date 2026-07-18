'use client';

import {
    IconArrowRight,
    IconBrandDocker,
    IconCheck,
    IconExternalLink,
    IconKey,
    IconLoader2,
    IconMessage,
    IconRefresh,
    IconRocket,
    IconWorld,
} from '@tabler/icons-react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { usePageTitle } from '@/hooks/use-page-title';
import { apiAction, apiGet, apiPost } from '@/lib/api-client';

interface DockerStatus {
    installed: boolean;
    running: boolean;
    version?: string;
    error?: string;
    hint?: string;
    platform: string;
}

interface OnboardingStatus {
    completed: boolean;
    hasDocker: boolean;
    hasAuth: boolean;
    hasWorlds: boolean;
    hasAgents: boolean;
    docker: DockerStatus;
}

interface ProviderRow {
    provider: string;
    connected: boolean;
    source?: string;
    plan?: string;
}

type StepId = 'auth' | 'chat' | 'docker' | 'world';

const STEPS: { id: StepId; title: string; subtitle: string; icon: typeof IconBrandDocker }[] = [
    {
        id: 'docker',
        title: 'Connect Docker',
        subtitle: 'spwn runs every agent inside an isolated Docker container.',
        icon: IconBrandDocker,
    },
    {
        id: 'auth',
        title: 'Connect a model provider',
        subtitle: 'Sign in to Anthropic or OpenAI so your agents can think.',
        icon: IconKey,
    },
    {
        id: 'world',
        title: 'Spawn your first world',
        subtitle: 'A world is an isolated workspace where one or more agents live.',
        icon: IconWorld,
    },
    {
        id: 'chat',
        title: 'Talk to an agent',
        subtitle: 'Send your first message and watch it work.',
        icon: IconMessage,
    },
];

export default function WelcomePage() {
    usePageTitle('Welcome');
    const router = useRouter();
    const [status, setStatus] = useState<null | OnboardingStatus>(null);
    const [active, setActive] = useState<StepId>('docker');
    const [refreshing, setRefreshing] = useState(false);
    const [finishing, setFinishing] = useState(false);

    const fetchStatus = useCallback(async () => {
        try {
            const s = await apiGet<OnboardingStatus>('/api/system/onboarding');
            setStatus(s);
            // Auto-advance to first incomplete step on initial load.
            setActive((prev) => {
                if (prev !== 'docker' && completed(prev, s)) {
                    return nextStep(prev);
                }
                if (!s.hasDocker) {
                    return 'docker';
                }
                if (!s.hasAuth) {
                    return 'auth';
                }
                if (!s.hasWorlds) {
                    return 'world';
                }
                return 'chat';
            });
        } catch {
            // API not reachable yet - render the page anyway with empty state.
        }
    }, []);

    useEffect(() => {
        fetchStatus();
        const id = setInterval(fetchStatus, 4000);
        return () => clearInterval(id);
    }, [fetchStatus]);

    // If the wizard was already completed, send the user straight to the
    // Worlds page. They can re-trigger the wizard manually from settings.
    useEffect(() => {
        if (status?.completed) {
            router.replace('/');
        }
    }, [status?.completed, router]);

    const handleRefresh = async () => {
        setRefreshing(true);
        try {
            await fetchStatus();
        } finally {
            setRefreshing(false);
        }
    };

    const handleFinish = async () => {
        setFinishing(true);
        try {
            await apiPost('/api/system/onboarding/complete');
            router.replace('/');
        } finally {
            setFinishing(false);
        }
    };

    const allDone = status?.hasDocker && status?.hasAuth && status?.hasWorlds;

    return (
        <div className="min-h-full overflow-y-auto px-6 py-10">
            <div className="mx-auto max-w-3xl">
                <header className="mb-10 text-center">
                    <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 text-[10px] uppercase tracking-wider text-muted-foreground/70">
                        <IconRocket size={11} />
                        First-run setup
                    </div>
                    <h1 className="font-heading text-3xl tracking-wide text-foreground/90">
                        Welcome to spwn
                    </h1>
                    <p className="mx-auto mt-2 max-w-lg text-sm text-muted-foreground/70">
                        The control plane for AI agents. Four quick steps and you&apos;re ready to
                        spawn your first world.
                    </p>
                </header>

                <ol className="space-y-3">
                    {STEPS.map((step, idx) => {
                        const done = status ? completed(step.id, status) : false;
                        const isActive = active === step.id;
                        const Icon = step.icon;
                        let liClass: string;
                        if (isActive) {
                            liClass = 'border-white/15 bg-white/[0.04]';
                        } else if (done) {
                            liClass = 'border-emerald-500/20 bg-emerald-500/[0.03]';
                        } else {
                            liClass = 'border-white/[0.06] bg-white/[0.015]';
                        }
                        let circleClass: string;
                        if (done) {
                            circleClass =
                                'border-emerald-400/40 bg-emerald-500/15 text-emerald-300';
                        } else if (isActive) {
                            circleClass = 'border-white/20 bg-white/[0.06] text-foreground/80';
                        } else {
                            circleClass =
                                'border-white/10 bg-white/[0.02] text-muted-foreground/50';
                        }
                        let statusLabel: string;
                        if (done) {
                            statusLabel = 'Done';
                        } else if (isActive) {
                            statusLabel = 'Active';
                        } else {
                            statusLabel = 'Pending';
                        }
                        return (
                            <li
                                className={`overflow-hidden rounded-xl border transition-colors ${liClass}`}
                                key={step.id}
                            >
                                <button
                                    className="flex w-full items-center gap-4 px-4 py-3 text-left"
                                    onClick={() => setActive(step.id)}
                                    type="button"
                                >
                                    <div
                                        className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full border ${circleClass}`}
                                    >
                                        {done ? <IconCheck size={14} /> : <Icon size={14} />}
                                    </div>
                                    <div className="min-w-0 flex-1">
                                        <div className="flex items-center gap-2">
                                            <span className="text-[10px] tabular-nums text-muted-foreground/40">
                                                {String(idx + 1).padStart(2, '0')}
                                            </span>
                                            <span
                                                className={`text-sm font-medium ${
                                                    done
                                                        ? 'text-emerald-200/90'
                                                        : 'text-foreground/90'
                                                }`}
                                            >
                                                {step.title}
                                            </span>
                                        </div>
                                        <p className="mt-0.5 text-xs text-muted-foreground/60">
                                            {step.subtitle}
                                        </p>
                                    </div>
                                    <span className="text-[10px] uppercase tracking-wider text-muted-foreground/40">
                                        {statusLabel}
                                    </span>
                                </button>

                                {isActive && (
                                    <div className="border-t border-white/[0.06] px-4 py-4">
                                        <StepBody
                                            onAdvance={() => setActive(nextStep(step.id))}
                                            onRefresh={handleRefresh}
                                            refreshing={refreshing}
                                            status={status}
                                            stepId={step.id}
                                        />
                                    </div>
                                )}
                            </li>
                        );
                    })}
                </ol>

                <div className="mt-8 flex items-center justify-between border-t border-white/[0.06] pt-6">
                    <button
                        className="text-xs text-muted-foreground/60 hover:text-foreground/80"
                        onClick={() => router.replace('/')}
                        type="button"
                    >
                        Skip for now
                    </button>
                    <button
                        className="inline-flex items-center gap-2 rounded-lg border border-emerald-400/30 bg-emerald-500/10 px-4 py-2 text-xs font-medium text-emerald-100 transition-colors hover:bg-emerald-500/20 disabled:cursor-not-allowed disabled:border-white/10 disabled:bg-white/[0.02] disabled:text-muted-foreground/40"
                        disabled={!allDone || finishing}
                        onClick={handleFinish}
                        type="button"
                    >
                        {finishing ? (
                            <IconLoader2 className="animate-spin" size={12} />
                        ) : (
                            <IconCheck size={12} />
                        )}
                        Finish setup
                    </button>
                </div>
            </div>
        </div>
    );
}

function StepBody({
    stepId,
    status,
    onRefresh,
    refreshing,
    onAdvance,
}: {
    stepId: StepId;
    status: null | OnboardingStatus;
    onRefresh: () => void;
    refreshing: boolean;
    onAdvance: () => void;
}) {
    if (stepId === 'docker') {
        return (
            <DockerStep
                onAdvance={onAdvance}
                onRefresh={onRefresh}
                refreshing={refreshing}
                status={status}
            />
        );
    }
    if (stepId === 'auth') {
        return <AuthStep onAdvance={onAdvance} status={status} />;
    }
    if (stepId === 'world') {
        return <WorldStep onAdvance={onAdvance} status={status} />;
    }
    return <ChatStep />;
}

function DockerStep({
    status,
    onRefresh,
    refreshing,
    onAdvance,
}: {
    status: null | OnboardingStatus;
    onRefresh: () => void;
    refreshing: boolean;
    onAdvance: () => void;
}) {
    const docker = status?.docker;
    const ok = docker?.installed && docker?.running;
    const installUrl =
        docker?.platform === 'linux'
            ? 'https://docs.docker.com/engine/install/'
            : 'https://www.docker.com/products/docker-desktop/';

    return (
        <div className="space-y-3">
            <div
                className={`rounded-lg border px-3 py-2.5 text-xs ${
                    ok
                        ? 'border-emerald-400/30 bg-emerald-500/[0.06] text-emerald-100'
                        : 'border-red-400/30 bg-red-500/[0.06] text-red-100'
                }`}
            >
                <div className="flex items-center gap-2">
                    <IconBrandDocker size={14} />
                    <span className="font-medium">
                        {(() => {
                            if (ok) {
                                return `Docker ${docker?.version ? `v${docker.version}` : ''} running`;
                            }
                            if (!docker?.installed) {
                                return 'Docker is not installed on this machine';
                            }
                            return 'Docker is installed but the daemon is not running';
                        })()}
                    </span>
                </div>
                {!ok && docker?.hint && (
                    <p className="mt-1 ml-[22px] text-[11px] opacity-80">{docker.hint}</p>
                )}
            </div>

            {!ok && (
                <div className="flex flex-wrap items-center gap-2">
                    {!docker?.installed && (
                        <a
                            className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-[11px] font-medium text-foreground/90 hover:bg-white/[0.08]"
                            href={installUrl}
                            rel="noreferrer"
                            target="_blank"
                        >
                            Install Docker
                            <IconExternalLink size={11} />
                        </a>
                    )}
                    <button
                        className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-[11px] font-medium text-foreground/90 hover:bg-white/[0.08] disabled:opacity-50"
                        disabled={refreshing}
                        onClick={onRefresh}
                    >
                        <IconRefresh className={refreshing ? 'animate-spin' : ''} size={11} />
                        Recheck
                    </button>
                </div>
            )}

            {ok && (
                <button
                    className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-[11px] font-medium text-foreground/90 hover:bg-white/[0.08]"
                    onClick={onAdvance}
                >
                    Continue <IconArrowRight size={11} />
                </button>
            )}
        </div>
    );
}

const reconnect = async (provider: string) => {
    await apiAction('/api/auth/reconnect', { provider });
};

function AuthStep({
    status,
    onAdvance,
}: {
    status: null | OnboardingStatus;
    onAdvance: () => void;
}) {
    const [providers, setProviders] = useState<null | ProviderRow[]>(null);

    useEffect(() => {
        apiGet<{ providers: ProviderRow[] }>('/api/auth/providers')
            .then((d) => setProviders(d.providers ?? []))
            .catch(() => setProviders([]));
    }, [status?.hasAuth]);

    return (
        <div className="space-y-3">
            <p className="text-[11px] text-muted-foreground/60">
                spwn supports subscription sign-in (Anthropic / Codex) and API keys via env vars.
            </p>
            <div className="space-y-1.5">
                {providers === null && (
                    <div className="rounded-md border border-white/[0.06] bg-white/[0.015] px-3 py-2 text-[11px] text-muted-foreground/50">
                        Loading providers…
                    </div>
                )}
                {providers?.map((p) => (
                    <div
                        className="flex items-center justify-between rounded-md border border-white/[0.06] bg-white/[0.015] px-3 py-2"
                        key={p.provider}
                    >
                        <div className="flex items-center gap-2">
                            <div
                                className={`h-1.5 w-1.5 rounded-full ${
                                    p.connected ? 'bg-emerald-400' : 'bg-muted-foreground/30'
                                }`}
                            />
                            <span className="text-xs font-medium capitalize text-foreground/90">
                                {p.provider}
                            </span>
                            {p.source && (
                                <span className="text-[10px] text-muted-foreground/40">
                                    {p.source}
                                </span>
                            )}
                        </div>
                        {!p.connected && (
                            <button
                                className="text-[11px] text-foreground/70 hover:text-foreground"
                                onClick={() => reconnect(p.provider)}
                            >
                                Connect
                            </button>
                        )}
                        {p.connected && (
                            <span className="text-[10px] uppercase tracking-wider text-emerald-300/80">
                                Connected
                            </span>
                        )}
                    </div>
                ))}
            </div>
            {status?.hasAuth && (
                <button
                    className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-[11px] font-medium text-foreground/90 hover:bg-white/[0.08]"
                    onClick={onAdvance}
                >
                    Continue <IconArrowRight size={11} />
                </button>
            )}
        </div>
    );
}

function WorldStep({
    status,
    onAdvance,
}: {
    status: null | OnboardingStatus;
    onAdvance: () => void;
}) {
    return (
        <div className="space-y-3">
            <p className="text-[11px] text-muted-foreground/60">
                Open the worlds page and click{' '}
                <span className="text-foreground/80">Spawn world</span>. We&apos;ll detect it as
                soon as a world is created.
            </p>
            <div className="flex items-center gap-2">
                <Link
                    className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-[11px] font-medium text-foreground/90 hover:bg-white/[0.08]"
                    href="/"
                >
                    Open worlds
                    <IconArrowRight size={11} />
                </Link>
                {status?.hasWorlds && (
                    <span className="inline-flex items-center gap-1 text-[11px] text-emerald-300/80">
                        <IconCheck size={11} />
                        World detected
                    </span>
                )}
            </div>
            {status?.hasWorlds && (
                <button
                    className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-[11px] font-medium text-foreground/90 hover:bg-white/[0.08]"
                    onClick={onAdvance}
                >
                    Continue <IconArrowRight size={11} />
                </button>
            )}
        </div>
    );
}

function ChatStep() {
    return (
        <div className="space-y-3">
            <p className="text-[11px] text-muted-foreground/60">
                Open any world from the sidebar and send a message in the chat panel. Once you
                finish here, click <span className="text-foreground/80">Finish setup</span> below.
            </p>
            <Link
                className="inline-flex items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.04] px-3 py-1.5 text-[11px] font-medium text-foreground/90 hover:bg-white/[0.08]"
                href="/"
            >
                Go to worlds <IconArrowRight size={11} />
            </Link>
        </div>
    );
}

function completed(step: StepId, status: OnboardingStatus): boolean {
    switch (step) {
        case 'docker': {
            return status.hasDocker;
        }
        case 'auth': {
            return status.hasAuth;
        }
        case 'world': {
            return status.hasWorlds;
        }
        case 'chat': {
            return false;
        } // User explicitly finishes
    }
}

function nextStep(step: StepId): StepId {
    const order: StepId[] = ['docker', 'auth', 'world', 'chat'];
    const i = order.indexOf(step);
    return order[Math.min(i + 1, order.length - 1)];
}
