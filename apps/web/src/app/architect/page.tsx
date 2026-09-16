'use client';

import {
    IconHexagonFilled,
    IconMessageCircle,
    IconPlayerPlay,
    IconPlayerStop,
    IconRefresh,
    IconTerminal2,
} from '@tabler/icons-react';
import { useEffect, useRef, useState } from 'react';

import { goApiUrl } from '@/api/client';
import { ActionButton } from '@/components/action-button';
import { Chat } from '@/components/chat';
import type { ChatBubble } from '@/components/chat';
import { MetricGrid } from '@/components/ds';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { ProgressShimmer } from '@/components/progress-shimmer';
import { Skeleton } from '@/components/ui/skeleton';
import { useArchitectChat } from '@/contexts/architect-chat-context';
import { usePageTitle } from '@/hooks/use-page-title';
import { useProgressMessages } from '@/hooks/use-progress-messages';

// ── Architect States ────────────────────────────────────────────────────

type ArchitectState = 'offline' | 'running' | 'starting' | 'stopping';

function deriveState(isRunning: boolean, actionLoading: null | string): ArchitectState {
    if (actionLoading === 'start') {
        return 'starting';
    }
    if (actionLoading === 'stop') {
        return 'stopping';
    }
    return isRunning ? 'running' : 'offline';
}

function describeState(s: ArchitectState): string {
    if (s === 'running') {
        return 'Running. Talk to it in natural language.';
    }
    if (s === 'starting') {
        return 'Starting...';
    }
    if (s === 'stopping') {
        return 'Stopping...';
    }
    return 'Offline';
}

// ── Offline State ───────────────────────────────────────────────────────

function OfflineView({ onStart, disabled }: { onStart: () => void; disabled: boolean }) {
    return (
        <div className="-mt-12 flex flex-1 items-center justify-center">
            <div className="flex max-w-md flex-col items-center text-center">
                <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.08] bg-white/[0.04]">
                    <IconHexagonFilled className="text-muted-foreground/20" size={28} />
                </div>

                <h2 className="font-heading text-foreground/70 mb-2 text-lg tracking-wide">
                    Architect is offline
                </h2>
                <p className="text-muted-foreground/40 mb-8 text-sm leading-relaxed">
                    The Architect runs in the background and manages everything for you: creating
                    agents, spawning worlds, and keeping track of tasks. Start it to begin working.
                </p>

                <button
                    className="group flex items-center gap-3 rounded-xl border border-white/[0.10] bg-white/[0.06] px-6 py-3 transition-all duration-200 hover:border-white/[0.16] hover:bg-white/[0.10] disabled:opacity-40"
                    disabled={disabled}
                    onClick={onStart}
                >
                    <IconPlayerPlay
                        className="text-green-400/80 group-hover:text-green-400"
                        size={18}
                    />
                    <span className="text-foreground/70 group-hover:text-foreground/90 text-sm font-medium">
                        Start Architect
                    </span>
                </button>

                <div className="text-muted-foreground/25 mt-6 flex items-center gap-2 font-mono text-[11px]">
                    <IconTerminal2 size={13} />
                    <span>spwn architect start</span>
                </div>
            </div>
        </div>
    );
}

// ── Starting State ──────────────────────────────────────────────────────

interface SpawnProgress {
    event: string;
    detail: string;
    inProgress: boolean;
    startedAt?: string;
    finishedAt?: string;
    elapsedSeconds: number;
    error?: string;
    containerId?: string;
    logTail?: string;
}

// Friendly labels for the stable event keys emitted by
// Universe.StartArchitectDaemonWithOpts. Anything not in this map
// Falls back to a humanised version of the key + the raw detail.
const PROGRESS_LABELS: Record<string, string> = {
    queued: 'Preparing to start architect…',
    docker_check: 'Connecting to Docker…',
    cleanup: 'Cleaning up old container…',
    image_resolve: 'Resolving architect image…',
    image_building: 'Building architect image (first run takes minutes)…',
    image_ready: 'Image ready',
    credentials_sync: 'Syncing credentials…',
    host_files: 'Preparing host files…',
    container_creating: 'Creating container…',
    container_starting: 'Starting container…',
    ready: 'Architect is ready',
    already_running: 'Architect already running',
};

function describeProgress(p: null | SpawnProgress): string {
    if (!p) {
        return 'Sending start signal…';
    }
    const label = PROGRESS_LABELS[p.event] ?? p.event.replaceAll('_', ' ');
    if (p.detail) {
        return `${label} - ${p.detail}`;
    }
    return label;
}

function StartingView({
    progress,
    fallbackMessage,
    showLogs,
    onToggleLogs,
}: {
    progress: null | SpawnProgress;
    fallbackMessage: string;
    showLogs: boolean;
    onToggleLogs: () => void;
}) {
    const message = progress ? describeProgress(progress) : fallbackMessage;
    const elapsed = progress?.elapsedSeconds ?? 0;
    const errored = Boolean(progress?.error);

    return (
        <div className="-mt-12 flex flex-1 items-center justify-center px-4">
            <div className="flex w-full max-w-lg flex-col items-center text-center">
                <div
                    className={`relative mb-6 flex h-16 w-16 items-center justify-center rounded-2xl border ${
                        errored
                            ? 'border-red-400/30 bg-red-500/[0.06]'
                            : 'border-white/[0.08] bg-white/[0.04]'
                    }`}
                >
                    <IconHexagonFilled
                        className={errored ? 'text-red-400/60' : 'animate-pulse text-yellow-400/40'}
                        size={28}
                    />
                </div>

                <h2 className="font-heading text-foreground/70 mb-2 text-lg tracking-wide">
                    {errored ? 'Architect failed to start' : 'Starting Architect'}
                </h2>
                <p
                    className={`mb-4 text-sm leading-relaxed ${errored ? 'text-red-300/70' : 'text-muted-foreground/50'}`}
                >
                    {errored ? progress?.error : message}
                </p>

                {!errored && (
                    <div className="mb-3 w-48">
                        <ProgressShimmer active message="" />
                    </div>
                )}

                <div className="text-muted-foreground/40 flex items-center gap-2 text-[10px] tracking-wider uppercase">
                    <span>{elapsed}s elapsed</span>
                    <span className="opacity-30">·</span>
                    <button
                        className="hover:text-foreground/80 transition-colors"
                        onClick={onToggleLogs}
                        type="button"
                    >
                        {showLogs ? 'Hide logs' : 'View logs'}
                    </button>
                </div>

                {showLogs && progress?.logTail && (
                    <pre className="text-muted-foreground/70 mt-4 max-h-64 w-full overflow-auto rounded-lg border border-white/[0.06] bg-black/40 px-3 py-2 text-left font-mono text-[10px] leading-snug">
                        {progress.logTail}
                    </pre>
                )}
            </div>
        </div>
    );
}

// ── Main Page ───────────────────────────────────────────────────────────

export default function ArchitectPage() {
    usePageTitle('Architect');

    const {
        messages,
        chatInput,
        setChatInput,
        sending,
        sendMessage,
        architectStatus,
        isRunning,
        highlightTitle: _highlightTitle,
        setArchitectStatus,
        loading,
    } = useArchitectChat();

    const [actionLoading, setActionLoading] = useState<null | string>(null);
    const [feedback, setFeedback] = useState<null | string>(null);
    const [startPolling, setStartPolling] = useState(false);
    const [spawnProgress, setSpawnProgress] = useState<null | SpawnProgress>(null);
    const [showSpawnLogs, setShowSpawnLogs] = useState(false);
    const pollRef = useRef<null | ReturnType<typeof setInterval>>(null);

    const state = deriveState(isRunning, actionLoading);

    // Fallback copy used only until the first /api/architect/status
    // Poll returns. Real progress is then driven by spawnProgress.
    const startFallbackMessage = 'Sending start signal…';

    const stopProgressMessage = useProgressMessages(state === 'stopping', [
        { after: 0, text: 'Stopping architect...' },
        { after: 5, text: 'Shutting down container...' },
        { after: 15, text: 'Cleaning up...' },
    ]);

    const bubbles: ChatBubble[] = messages.map((m) => ({
        role: m.role === 'architect' ? 'assistant' : 'user',
        blocks: m.blocks,
        content: m.content,
        timestamp: m.timestamp,
        error: m.error,
        cost: m.cost,
        duration: m.duration,
    }));

    const showFeedback = (msg: string) => {
        setFeedback(msg);
        setTimeout(() => setFeedback(null), 4000);
    };

    // Poll for architect status after starting. The status endpoint
    // Now carries a `progress` field with the latest event, detail,
    // Log tail and final error from the universe spawn pipeline, so
    // We feed it straight into spawnProgress for the StartingView.
    useEffect(() => {
        if (!startPolling) {
            return;
        }
        pollRef.current = setInterval(async () => {
            try {
                const res = await fetch(goApiUrl('/api/architect/status'));
                if (!res.ok) {
                    return;
                }
                const data = await res.json();
                if (data.progress) {
                    setSpawnProgress(data.progress as SpawnProgress);
                    if (data.progress.error) {
                        // Stop polling on error so the failure message stays put.
                        setStartPolling(false);
                        setActionLoading(null);
                        return;
                    }
                }
                if (data.status === 'running') {
                    setStartPolling(false);
                    setActionLoading(null);
                    setSpawnProgress(null);
                    setShowSpawnLogs(false);
                    setArchitectStatus((s) => (s ? { ...s, status: 'running' } : s));
                    showFeedback('Architect started');
                }
            } catch {
                // Ignore polling errors
            }
        }, 2000);
        return () => {
            if (pollRef.current) {
                clearInterval(pollRef.current);
            }
        };
    }, [startPolling, setArchitectStatus]);

    const handleStart = async () => {
        setActionLoading('start');
        try {
            const res = await fetch(goApiUrl('/api/architect/start'), { method: 'POST' });
            if (res.ok) {
                setStartPolling(true);
            } else {
                const data = await res.json().catch(() => ({ error: 'Unknown error' }));
                showFeedback(`Error: ${data.error}`);
                setActionLoading(null);
            }
        } catch {
            showFeedback('Error: Failed to connect to API');
            setActionLoading(null);
        }
    };

    const handleStop = async () => {
        setActionLoading('stop');
        try {
            const res = await fetch(goApiUrl('/api/architect/stop'), { method: 'POST' });
            if (res.ok) {
                showFeedback('Architect stopped');
                setArchitectStatus((s) => (s ? { ...s, status: 'stopped' } : s));
            } else {
                const data = await res.json().catch(() => ({ error: 'Unknown error' }));
                showFeedback(`Error: ${data.error}`);
            }
        } catch {
            showFeedback('Error: Failed to connect to API');
        } finally {
            setActionLoading(null);
        }
    };

    const handleSendMessage = () => {
        void sendMessage();
    };

    const kpis = architectStatus?.kpis;

    const renderHeaderActions = () => {
        if (state === 'running') {
            return (
                <>
                    <ActionButton
                        compact
                        disabled={actionLoading !== null}
                        icon={<IconRefresh size={16} stroke={2.2} />}
                        label="Restart"
                        onClick={handleStart}
                    />
                    <ActionButton
                        compact
                        danger
                        disabled={actionLoading !== null}
                        icon={<IconPlayerStop size={16} stroke={2.2} />}
                        label="Stop"
                        onClick={handleStop}
                    />
                </>
            );
        }
        if (state === 'offline') {
            return (
                <ActionButton
                    compact
                    disabled={actionLoading !== null}
                    icon={<IconPlayerPlay size={16} stroke={2.2} />}
                    label="Start"
                    onClick={handleStart}
                />
            );
        }
        return null;
    };

    return (
        <Page>
            <PageHeader
                actions={renderHeaderActions()}
                description={describeState(state)}
                title="Architect"
            />

            {/* Feedback toast */}
            {feedback && (
                <div
                    className={`animate-in fade-in slide-in-from-top-2 rounded-lg px-4 py-2 font-mono text-xs duration-200 ${
                        feedback.startsWith('Error')
                            ? 'border border-red-500/20 bg-red-500/10 text-red-400'
                            : 'border border-green-500/20 bg-green-500/10 text-green-400'
                    }`}
                >
                    {feedback}
                </div>
            )}

            {/* ── Offline ── */}
            {state === 'offline' && !loading && (
                <OfflineView disabled={actionLoading !== null} onStart={handleStart} />
            )}

            {/* ── Starting ── */}
            {state === 'starting' && (
                <StartingView
                    fallbackMessage={startFallbackMessage}
                    onToggleLogs={() => setShowSpawnLogs((v) => !v)}
                    progress={spawnProgress}
                    showLogs={showSpawnLogs}
                />
            )}

            {/* ── Stopping ── */}
            {state === 'stopping' && (
                <div className="-mt-12 flex flex-1 items-center justify-center">
                    <div className="flex flex-col items-center text-center">
                        <p className="text-muted-foreground/50 mb-4 text-sm">
                            {stopProgressMessage}
                        </p>
                        <div className="w-48">
                            <ProgressShimmer active message="" />
                        </div>
                    </div>
                </div>
            )}

            {/* ── Loading (initial) ── */}
            {loading && state === 'offline' && (
                <div className="-mt-12 flex flex-1 items-center justify-center">
                    <div className="flex flex-col items-center gap-4">
                        <Skeleton className="h-16 w-16 rounded-2xl" />
                        <Skeleton className="h-4 w-40" />
                        <Skeleton className="h-3 w-56" />
                    </div>
                </div>
            )}

            {/* ── Running ── */}
            {state === 'running' && (
                <>
                    {/* KPI Metrics */}
                    <MetricGrid
                        className="w-fit gap-x-10"
                        columns={2}
                        items={[
                            { label: 'Worlds', value: kpis?.worlds ?? 0 },
                            { label: 'Agents', value: kpis?.agents ?? 0 },
                        ]}
                    />

                    {/* Chat - fills remaining page height, input sticks to the bottom */}
                    <Chat
                        autoFocus
                        className="min-h-[320px] flex-1"
                        disabled={sending}
                        emptyState={
                            <div className="flex flex-col items-center justify-center text-center">
                                <IconMessageCircle
                                    className="text-muted-foreground/15 mb-3"
                                    size={28}
                                />
                                <p className="text-muted-foreground/30 text-sm">Ask anything</p>
                                <p className="text-muted-foreground/20 mt-1 max-w-sm text-[11px]">
                                    &quot;Create an agent for the API project&quot;,
                                    &quot;What&apos;s running?&quot;, &quot;Spawn a world for the
                                    frontend repo&quot;
                                </p>
                            </div>
                        }
                        input={chatInput}
                        messages={bubbles}
                        onInputChange={setChatInput}
                        onSend={handleSendMessage}
                        placeholder="Talk to the Architect..."
                        typingText="Architect is thinking..."
                    />
                </>
            )}
        </Page>
    );
}
