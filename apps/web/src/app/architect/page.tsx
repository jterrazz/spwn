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

import { ActionButton } from '@/components/action-button';
import { Chat, type ChatBubble } from '@/components/chat';
import { MetricGrid } from '@/components/ds';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { ProgressShimmer } from '@/components/progress-shimmer';
import { Skeleton } from '@/components/ui/skeleton';
import { useArchitectChat } from '@/contexts/architect-chat-context';
import { usePageTitle } from '@/hooks/use-page-title';
import { useProgressMessages } from '@/hooks/use-progress-messages';
import { goApiUrl } from '@/lib/api-client';

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

// ── Offline State ───────────────────────────────────────────────────────

function OfflineView({ onStart, disabled }: { onStart: () => void; disabled: boolean }) {
    return (
        <div className="flex-1 flex items-center justify-center -mt-12">
            <div className="flex flex-col items-center text-center max-w-md">
                <div className="w-16 h-16 rounded-2xl bg-white/[0.04] border border-white/[0.08] flex items-center justify-center mb-6">
                    <IconHexagonFilled className="text-muted-foreground/20" size={28} />
                </div>

                <h2 className="text-lg font-heading tracking-wide text-foreground/70 mb-2">
                    Architect is offline
                </h2>
                <p className="text-sm text-muted-foreground/40 mb-8 leading-relaxed">
                    The Architect runs in the background and manages everything for you: creating
                    agents, spawning worlds, and keeping track of tasks. Start it to begin working.
                </p>

                <button
                    className="group flex items-center gap-3 px-6 py-3 rounded-xl bg-white/[0.06] border border-white/[0.10] hover:bg-white/[0.10] hover:border-white/[0.16] transition-all duration-200 disabled:opacity-40"
                    disabled={disabled}
                    onClick={onStart}
                >
                    <IconPlayerPlay
                        className="text-green-400/80 group-hover:text-green-400"
                        size={18}
                    />
                    <span className="text-sm font-medium text-foreground/70 group-hover:text-foreground/90">
                        Start Architect
                    </span>
                </button>

                <div className="mt-6 flex items-center gap-2 text-[11px] text-muted-foreground/25 font-mono">
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
    const label = PROGRESS_LABELS[p.event] ?? p.event.replace(/_/g, ' ');
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
        <div className="flex-1 flex items-center justify-center -mt-12 px-4">
            <div className="flex flex-col items-center text-center max-w-lg w-full">
                <div
                    className={`w-16 h-16 rounded-2xl border flex items-center justify-center mb-6 relative ${
                        errored
                            ? 'bg-red-500/[0.06] border-red-400/30'
                            : 'bg-white/[0.04] border-white/[0.08]'
                    }`}
                >
                    <IconHexagonFilled
                        className={errored ? 'text-red-400/60' : 'text-yellow-400/40 animate-pulse'}
                        size={28}
                    />
                </div>

                <h2 className="text-lg font-heading tracking-wide text-foreground/70 mb-2">
                    {errored ? 'Architect failed to start' : 'Starting Architect'}
                </h2>
                <p
                    className={`text-sm leading-relaxed mb-4 ${errored ? 'text-red-300/70' : 'text-muted-foreground/50'}`}
                >
                    {errored ? progress?.error : message}
                </p>

                {!errored && (
                    <div className="w-48 mb-3">
                        <ProgressShimmer active message="" />
                    </div>
                )}

                <div className="flex items-center gap-2 text-[10px] uppercase tracking-wider text-muted-foreground/40">
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
                    <pre className="mt-4 w-full max-h-64 overflow-auto rounded-lg border border-white/[0.06] bg-black/40 px-3 py-2 text-left text-[10px] leading-snug font-mono text-muted-foreground/70">
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

    return (
        <Page>
            <PageHeader
                actions={
                    state === 'running' ? (
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
                    ) : state === 'offline' ? (
                        <ActionButton
                            compact
                            disabled={actionLoading !== null}
                            icon={<IconPlayerPlay size={16} stroke={2.2} />}
                            label="Start"
                            onClick={handleStart}
                        />
                    ) : null
                }
                description={
                    state === 'running'
                        ? 'Running. Talk to it in natural language.'
                        : state === 'starting'
                          ? 'Starting...'
                          : state === 'stopping'
                            ? 'Stopping...'
                            : 'Offline'
                }
                title="Architect"
            />

            {/* Feedback toast */}
            {feedback && (
                <div
                    className={`px-4 py-2 rounded-lg text-xs font-mono animate-in fade-in slide-in-from-top-2 duration-200 ${
                        feedback.startsWith('Error')
                            ? 'bg-red-500/10 border border-red-500/20 text-red-400'
                            : 'bg-green-500/10 border border-green-500/20 text-green-400'
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
                <div className="flex-1 flex items-center justify-center -mt-12">
                    <div className="flex flex-col items-center text-center">
                        <p className="text-sm text-muted-foreground/50 mb-4">
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
                <div className="flex-1 flex items-center justify-center -mt-12">
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
                        className="flex-1 min-h-[320px]"
                        disabled={sending}
                        emptyState={
                            <div className="flex flex-col items-center justify-center text-center">
                                <IconMessageCircle
                                    className="text-muted-foreground/15 mb-3"
                                    size={28}
                                />
                                <p className="text-sm text-muted-foreground/30">Ask anything</p>
                                <p className="text-[11px] text-muted-foreground/20 mt-1 max-w-sm">
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
