'use client';

import {
    IconAlertTriangle,
    IconArrowRight,
    IconCamera,
    IconDownload,
    IconFileText,
    IconLoader2,
    IconPencil,
    IconPlayerPlay,
    IconRestore,
    IconSparkles,
    IconTrash,
    IconUserPlus,
    IconWorld,
    IconX,
} from '@tabler/icons-react';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useRef, useState } from 'react';

import { apiAction, apiDelete, apiGet, apiPost, goApiUrl } from '@/api/client';
import { ActionButton } from '@/components/action-button';
import { useRefetch } from '@/components/app-shell';
import {
    DataTable,
    StatusDot as DSStatusDot,
    MetricGrid,
    SectionHeader,
    Separator,
} from '@/components/ds';
import { PageHeader } from '@/components/page-header';
import { ProgressShimmer } from '@/components/progress-shimmer';
import { useToast } from '@/components/toast-provider';
import { Skeleton } from '@/components/ui/skeleton';
import { WorldPlanet } from '@/components/world-planet';
import { getWorkspaceSummary, getWorldName } from '@/domain/model';
import type { Agent, World } from '@/domain/model';
import { usePageTitle } from '@/hooks/use-page-title';
import { useProgressMessages } from '@/hooks/use-progress-messages';
import { ROLE_BADGE } from '@/styles/status-colors';

function timeAgo(iso: string): string {
    const d = Date.now() - new Date(iso).getTime();
    const m = Math.floor(d / 60_000);
    if (m < 60) {
        return `${m}m ago`;
    }
    const h = Math.floor(m / 60);
    if (h < 24) {
        return `${h}h ago`;
    }
    return `${Math.floor(h / 24)}d ago`;
}

const LOG_LEVEL_COLORS: Record<string, string> = {
    info: 'text-blue-400/70',
    warn: 'text-yellow-400/70',
    error: 'text-red-400/70',
    debug: 'text-muted-foreground/40',
};

type Panel = 'logs' | 'snapshots' | null;

// Hoisted out of the dashboard component so cell renderers are stable across renders.
const AGENT_COLUMNS = [
    {
        key: 'name',
        label: 'Name',
        width: '1fr',
        render: (a: Agent) => (
            <span className="text-foreground/85 truncate font-mono text-[13px]">{a.name}</span>
        ),
    },
    {
        key: 'role',
        label: 'Role',
        width: '80px',
        render: (a: Agent) => {
            const badge = ROLE_BADGE[a.role] ?? ROLE_BADGE.default;
            return (
                <span
                    className={`rounded border px-1.5 py-0.5 font-mono text-[9px] tracking-wider uppercase ${badge}`}
                >
                    {a.role}
                </span>
            );
        },
    },
    {
        key: 'status',
        label: 'Status',
        width: '100px',
        render: (a: Agent) => (
            <span className="flex items-center gap-1.5">
                <DSStatusDot status={a.status} />
                <span className="text-muted-foreground/50 font-mono text-[11px] capitalize">
                    {a.status}
                </span>
            </span>
        ),
    },
];

export default function WorldDashboard() {
    const params = useParams();
    const router = useRouter();
    const worldId = params.id as string;
    const [world, setWorld] = useState<null | World>(null);
    const [loading, setLoading] = useState(true);
    const [activePanel, setActivePanel] = useState<Panel>(null);
    const [showDestroyConfirm, setShowDestroyConfirm] = useState(false);
    const [actionFeedback, setActionFeedback] = useState<null | string>(null);
    const [actionLoading, setActionLoading] = useState<null | string>(null);
    const [showRenameDialog, setShowRenameDialog] = useState(false);
    const [renameInput, setRenameInput] = useState('');
    const [renaming, setRenaming] = useState(false);

    const destroyProgressMessage = useProgressMessages(actionLoading === 'destroy', [
        { after: 0, text: 'Destroying world...' },
        { after: 5, text: 'Stopping containers...' },
        { after: 15, text: 'Cleaning up...' },
    ]);

    const { toast } = useToast();
    const refetchSidebar = useRefetch();
    const worldName = world ? getWorldName(world) : null;
    usePageTitle(worldName);

    const fetchWorld = useCallback(() => {
        apiGet<World[]>('/api/worlds')
            .then((worlds) => {
                const found = worlds.find((w) => w.id === worldId);
                setWorld(found ?? null);
                setLoading(false);
            })
            .catch(() => {
                setWorld(null);
                setLoading(false);
            });
    }, [worldId]);

    useEffect(() => {
        fetchWorld();
        const interval = setInterval(fetchWorld, 5000);
        return () => clearInterval(interval);
    }, [fetchWorld]);
    const callAction = async (goPath: string, body?: unknown) => {
        setActionLoading(goPath);
        try {
            const result = await apiAction(goPath, body);
            if (!result.ok) {
                showFeedback(`Error: ${result.error || 'Unknown error'}`);
                return false;
            }
            // Immediately refetch data after successful mutation
            fetchWorld();
            refetchSidebar();
            return true;
        } catch {
            showFeedback('Error: Failed to connect to API');
            return false;
        } finally {
            setActionLoading(null);
        }
    };

    const handleRename = async () => {
        setRenaming(true);
        try {
            const res = await fetch(goApiUrl(`/api/worlds/${worldId}`), {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: renameInput.trim() }),
            });
            if (!res.ok) {
                const data = await res.json().catch(() => ({}));
                showFeedback(`Error: ${data.error || 'Failed to rename'}`);
                setRenaming(false);
                return;
            }
            fetchWorld();
            refetchSidebar();
            setShowRenameDialog(false);
        } catch {
            showFeedback('Error: Failed to connect to API');
        } finally {
            setRenaming(false);
        }
    };

    // Snapshots are not yet available from the API
    const snapshots: {
        id: string;
        worldId: string;
        name: string;
        created_at: string;
        size: string;
        agents: number;
    }[] = [];

    // Event log state (was "logs" - now the per-world semantic event feed)
    const [logs, setLogs] = useState<
        { timestamp: string; level: string; source: string; message: string }[]
    >([]);
    const [logsLoading, setLogsLoading] = useState(false);
    const logsEndRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (activePanel !== 'logs') {
            return;
        }
        setLogsLoading(true);
        const controller = new AbortController();

        fetch(goApiUrl(`/api/activity?world=${worldId}&limit=200`), { signal: controller.signal })
            .then(async (res) => {
                if (!res.ok) {
                    throw new Error('Failed to fetch events');
                }
                const data = await res.json();
                const events = (data.events ?? []) as {
                    timestamp: string;
                    type: string;
                    phrase: string;
                    actor: string;
                }[];
                setLogs(
                    events.map((e) => ({
                        timestamp: e.timestamp,
                        level: 'info',
                        source: e.actor || 'world',
                        message: e.phrase || e.type,
                    })),
                );
            })
            .catch(() => {
                setLogs([]);
            })
            .finally(() => setLogsLoading(false));

        return () => controller.abort();
    }, [activePanel, worldId]);

    const showFeedback = (msg: string) => {
        const isError = msg.toLowerCase().startsWith('error');
        toast(msg, isError ? 'error' : 'success');
        setActionFeedback(msg);
        setTimeout(() => setActionFeedback(null), 2500);
    };

    if (loading) {
        return (
            <div className="space-y-8 p-8">
                <div className="flex items-center gap-4">
                    <Skeleton className="h-2.5 w-2.5 rounded-full" />
                    <div className="space-y-2">
                        <Skeleton className="h-7 w-32" />
                        <Skeleton className="h-3 w-48" />
                    </div>
                </div>
                <div className="flex gap-10">
                    {[1, 2, 3, 4].map((i) => (
                        <div key={i}>
                            <Skeleton className="mb-2 h-3 w-14" />
                            <Skeleton className="h-7 w-10" />
                        </div>
                    ))}
                </div>
                <div className="space-y-3">
                    <Skeleton className="h-4 w-24" />
                    <div className="grid grid-cols-2 gap-3">
                        {[1, 2].map((i) => (
                            <Skeleton className="h-16 rounded-xl" key={i} />
                        ))}
                    </div>
                </div>
            </div>
        );
    }

    if (!world) {
        return (
            <div className="p-8">
                <p className="text-muted-foreground/50">World not found</p>
            </div>
        );
    }

    const name = getWorldName(world);

    return (
        <div className="flex h-[calc(100vh-1px)] overflow-hidden">
            {/* Main content */}
            <div className="flex-1 space-y-6 overflow-y-auto px-6 pt-6 pb-12 md:space-y-8 md:px-8 md:pt-8">
                <PageHeader
                    actions={
                        <>
                            <ActionButton
                                compact
                                disabled={actionLoading !== null}
                                icon={<IconCamera size={16} stroke={2.2} />}
                                label="Snapshot"
                                onClick={async () => {
                                    const ok = await callAction(`/api/worlds/${worldId}/snapshot`);
                                    if (ok) {
                                        showFeedback('Snapshot saved!');
                                    }
                                }}
                            />
                            <ActionButton
                                compact
                                icon={<IconFileText size={16} stroke={2.2} />}
                                label="Logs"
                                onClick={() =>
                                    setActivePanel(activePanel === 'logs' ? null : 'logs')
                                }
                            />
                            <ActionButton
                                compact
                                icon={<IconRestore size={16} stroke={2.2} />}
                                label="Snapshots"
                                onClick={() =>
                                    setActivePanel(activePanel === 'snapshots' ? null : 'snapshots')
                                }
                            />
                            <ActionButton
                                compact
                                icon={<IconPencil size={16} stroke={2.2} />}
                                label="Rename"
                                onClick={() => {
                                    setRenameInput(world.name ?? '');
                                    setShowRenameDialog(true);
                                }}
                            />
                            <ActionButton
                                compact
                                danger
                                icon={<IconTrash size={16} stroke={2.2} />}
                                label="Destroy"
                                onClick={() => setShowDestroyConfirm(true)}
                            />
                        </>
                    }
                    description={`${world.config} · ${timeAgo(world.created_at)} · ${getWorkspaceSummary(world)}`}
                    leading={<WorldPlanet size="lg" world={world} />}
                    title={name}
                />

                {/* Rename dialog */}
                {showRenameDialog && (
                    <div className="fixed inset-0 z-50 flex items-center justify-center">
                        <div
                            className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                            onClick={() => !renaming && setShowRenameDialog(false)}
                        />
                        <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-md rounded-2xl border border-white/[0.08] p-6 shadow-2xl backdrop-blur-md">
                            <h3 className="font-heading text-foreground/90 mb-1 text-lg">
                                Rename World
                            </h3>
                            <p className="text-muted-foreground/50 mb-5 text-sm">
                                Leave empty to fall back to the auto-generated name (
                                <span className="font-mono">
                                    {world.id.split('-')[1] ?? world.id}
                                </span>
                                ).
                            </p>
                            <input
                                autoFocus
                                className="text-foreground/80 placeholder:text-muted-foreground/30 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none disabled:opacity-50"
                                disabled={renaming}
                                onChange={(e) => setRenameInput(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleRename();
                                    }
                                }}
                                placeholder="My Project"
                                type="text"
                                value={renameInput}
                            />
                            <div className="mt-6 flex justify-end gap-3">
                                <button
                                    className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-50"
                                    disabled={renaming}
                                    onClick={() => setShowRenameDialog(false)}
                                >
                                    Cancel
                                </button>
                                <button
                                    className="text-foreground/90 flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.1] px-4 py-2 text-sm transition-colors hover:bg-white/[0.16] disabled:opacity-50"
                                    disabled={renaming}
                                    onClick={handleRename}
                                >
                                    {renaming ? (
                                        <>
                                            <div className="border-foreground/30 border-t-foreground/80 h-3 w-3 animate-spin rounded-full border-2" />
                                            Saving…
                                        </>
                                    ) : (
                                        'Save'
                                    )}
                                </button>
                            </div>
                        </div>
                    </div>
                )}

                {/* Destroy confirmation */}
                {showDestroyConfirm && (
                    <div className="rounded-xl border border-red-500/30 bg-red-500/5 p-5">
                        <div className="flex items-start gap-3">
                            <IconAlertTriangle className="mt-0.5 shrink-0 text-red-400" size={20} />
                            <div className="flex-1">
                                <h3 className="font-heading text-sm text-red-300">
                                    Destroy World?
                                </h3>
                                <p className="mt-1 text-xs text-red-300/60">
                                    This will permanently destroy{' '}
                                    <span className="font-mono">{world.id}</span> and all its
                                    agents. This action cannot be undone.
                                </p>
                                {actionLoading === 'destroy' && (
                                    <ProgressShimmer
                                        active
                                        className="mt-2 mb-1"
                                        message={destroyProgressMessage}
                                    />
                                )}
                                <div className="mt-4 flex gap-2">
                                    <button
                                        className="rounded-lg border border-red-500/30 bg-red-500/20 px-4 py-2 text-xs text-red-300 transition-colors hover:bg-red-500/30 disabled:opacity-30"
                                        disabled={actionLoading !== null}
                                        onClick={async () => {
                                            setActionLoading('destroy');
                                            try {
                                                await apiDelete(`/api/worlds/${worldId}`);
                                                showFeedback('World destroyed');
                                                setShowDestroyConfirm(false);
                                                refetchSidebar();
                                                router.push('/');
                                            } catch {
                                                showFeedback('Error: Failed to destroy world');
                                                setShowDestroyConfirm(false);
                                            } finally {
                                                setActionLoading(null);
                                            }
                                        }}
                                    >
                                        {actionLoading === 'destroy' ? (
                                            <span className="flex items-center gap-2">
                                                <span className="h-3 w-3 animate-spin rounded-full border-2 border-red-300/40 border-t-red-300" />
                                                {destroyProgressMessage}
                                            </span>
                                        ) : (
                                            'Yes, destroy it'
                                        )}
                                    </button>
                                    <button
                                        className="text-muted-foreground/50 hover:text-foreground/70 rounded-lg px-4 py-2 text-xs transition-colors hover:bg-white/[0.04]"
                                        onClick={() => setShowDestroyConfirm(false)}
                                    >
                                        Cancel
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                )}

                {/* Action feedback toast */}
                {actionFeedback && (
                    <div className="animate-in fade-in slide-in-from-top-2 rounded-lg border border-green-500/20 bg-green-500/10 px-4 py-2 font-mono text-xs text-green-400 duration-200">
                        {actionFeedback}
                    </div>
                )}

                {/* Stats */}
                <MetricGrid
                    className="w-fit gap-x-10"
                    columns={4}
                    items={[
                        { label: 'Status', value: world.status },
                        { label: 'Agents', value: world.agents.length },
                        { label: 'Config', value: world.config },
                        { label: 'Uptime', value: timeAgo(world.created_at) },
                    ]}
                />

                <Separator />

                {/* Elements */}
                {world.manifest?.elements && world.manifest.elements.length > 0 && (
                    <>
                        <div>
                            <SectionHeader>Elements</SectionHeader>
                            <div className="flex flex-wrap gap-1.5">
                                {world.manifest.elements.map((el) => (
                                    <span
                                        className="text-foreground/60 border border-white/[0.06] bg-white/[0.04] px-2.5 py-1 font-mono text-[11px]"
                                        key={el}
                                    >
                                        {el}
                                    </span>
                                ))}
                            </div>
                        </div>
                        <Separator />
                    </>
                )}

                {/* Agents */}
                <div>
                    <div className="mb-3 flex items-center justify-between">
                        <SectionHeader className="mb-0">Agents</SectionHeader>
                        {world.agents.length > 0 && (
                            <button
                                className="text-muted-foreground/35 hover:text-foreground/70 font-mono text-[10px] transition-colors"
                                onClick={() => router.push('/agents')}
                            >
                                + Deploy
                            </button>
                        )}
                    </div>
                    {world.agents.length === 0 ? (
                        <EmptyAgentsView onDeployed={fetchWorld} worldId={worldId} />
                    ) : (
                        <DataTable<Agent>
                            columns={AGENT_COLUMNS}
                            emptyText="No agents deployed."
                            rowHref={(a) =>
                                `/agents/${encodeURIComponent(a.name)}?world=${worldId}`
                            }
                            rowKey={(a) => a.name}
                            rows={world.agents}
                        />
                    )}
                </div>
            </div>

            {/* ── Side panel for Logs/Snapshots ── */}
            {activePanel && (
                <div className="border-border/30 hidden w-96 shrink-0 flex-col overflow-hidden border-l md:flex">
                    <div className="border-border/30 flex shrink-0 items-center justify-between border-b px-5 py-4">
                        <h2 className="font-heading text-foreground/80 text-sm capitalize">
                            {activePanel}
                        </h2>
                        <button
                            className="text-muted-foreground/40 hover:text-foreground/70 transition-colors"
                            onClick={() => setActivePanel(null)}
                        >
                            <IconX size={16} />
                        </button>
                    </div>

                    <div className="flex-1 overflow-y-auto">
                        {activePanel === 'logs' && (
                            <div className="space-y-0.5 p-4 font-mono text-[11px]">
                                {logsLoading && logs.length === 0 && (
                                    <div className="text-muted-foreground/30 flex items-center justify-center gap-2 py-8">
                                        <div className="border-foreground/20 border-t-foreground/50 h-3 w-3 animate-spin rounded-full border-2" />
                                        <span className="text-sm">Connecting to log stream...</span>
                                    </div>
                                )}
                                {!logsLoading && logs.length === 0 && (
                                    <div className="py-8 text-center">
                                        <p className="text-muted-foreground/30 text-sm">
                                            No logs available
                                        </p>
                                        <p className="text-muted-foreground/20 mt-1 font-mono text-[10px]">
                                            Use the CLI: spwn logs {worldId}
                                        </p>
                                    </div>
                                )}
                                {logs.map((log) => (
                                    <div
                                        className="border-border/10 flex gap-2 border-b py-1.5 last:border-0"
                                        key={`${log.timestamp}-${log.source}-${log.message}`}
                                    >
                                        <span className="text-muted-foreground/25 w-14 shrink-0">
                                            {new Date(log.timestamp).toLocaleTimeString([], {
                                                hour: '2-digit',
                                                minute: '2-digit',
                                                second: '2-digit',
                                            })}
                                        </span>
                                        <span
                                            className={`w-10 shrink-0 uppercase ${LOG_LEVEL_COLORS[log.level]}`}
                                        >
                                            {log.level}
                                        </span>
                                        <span className="text-muted-foreground/40 w-16 shrink-0">
                                            {log.source}
                                        </span>
                                        <span className="text-foreground/60 break-all">
                                            {log.message}
                                        </span>
                                    </div>
                                ))}
                                <div ref={logsEndRef} />
                            </div>
                        )}

                        {activePanel === 'snapshots' && (
                            <div className="space-y-3 p-4">
                                {snapshots.length === 0 ? (
                                    <p className="text-muted-foreground/30 py-8 text-center text-sm">
                                        No snapshots
                                    </p>
                                ) : (
                                    snapshots.map((snap) => (
                                        <div className="glass-subtle p-4" key={snap.id}>
                                            <div className="mb-2 flex items-center justify-between">
                                                <span className="text-foreground/70 font-mono text-xs">
                                                    {snap.name}
                                                </span>
                                                <span className="text-muted-foreground/30 font-mono text-[10px]">
                                                    {snap.size}
                                                </span>
                                            </div>
                                            <p className="text-muted-foreground/40 mb-3 font-mono text-[10px]">
                                                {timeAgo(snap.created_at)} · {snap.agents} agent
                                                {snap.agents === 1 ? '' : 's'}
                                            </p>
                                            <div className="flex gap-2">
                                                <button
                                                    className="text-muted-foreground/50 hover:text-foreground/70 flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[10px] transition-colors hover:bg-white/[0.04]"
                                                    onClick={() =>
                                                        showFeedback(`Restoring "${snap.name}"...`)
                                                    }
                                                >
                                                    <IconPlayerPlay size={12} />
                                                    Restore
                                                </button>
                                                <button
                                                    className="text-muted-foreground/50 hover:text-foreground/70 flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[10px] transition-colors hover:bg-white/[0.04]"
                                                    onClick={() => showFeedback('Downloading...')}
                                                >
                                                    <IconDownload size={12} />
                                                    Export
                                                </button>
                                                <button
                                                    className="ml-auto flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[10px] text-red-400/50 transition-colors hover:bg-red-500/10 hover:text-red-400"
                                                    onClick={() => showFeedback('Snapshot deleted')}
                                                >
                                                    <IconTrash size={12} />
                                                    Delete
                                                </button>
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

// ── Empty-agents view ─────────────────────────────────────────────────
//
// Replaces the bare "No agents deployed" placeholder when a world has
// Zero members. Two paths to populate:
//
//   1. One of the user's already-installed agents → POST
//      /api/worlds/{id}/agents and refetch.
//   2. A fresh template from the gallery → POST /api/examples/<slug>
//      /install, then deploy the first installed agent that is
//      Compatible (same flow but two requests instead of one).
//
// The component is self-contained - fetches its own data, has its own
// Loading/error state, and reports back through onDeployed so the
// Parent can refetch the world.

interface InstalledAgentItem {
    name: string;
    layers?: Record<string, string[]>;
}

interface EmptyAgentsExample {
    slug: string;
    name: string;
    tagline: string;
    agents: string[];
    worlds: string[];
}

function EmptyAgentsView({ worldId, onDeployed }: { worldId: string; onDeployed: () => void }) {
    const [installed, setInstalled] = useState<InstalledAgentItem[] | null>(null);
    const [gallery, setGallery] = useState<EmptyAgentsExample[] | null>(null);
    const [busy, setBusy] = useState<null | string>(null);
    const [errorMessage, setErrorMessage] = useState<null | string>(null);
    const [showGallery, setShowGallery] = useState(false);

    useEffect(() => {
        apiGet<InstalledAgentItem[]>('/api/agents')
            .then((data) => setInstalled(data ?? []))
            .catch(() => setInstalled([]));
        apiGet<{ examples: EmptyAgentsExample[] }>('/api/examples')
            .then((data) => setGallery(data.examples ?? []))
            .catch(() => setGallery([]));
    }, []);

    const deploy = async (name: string, role: string = 'worker') => {
        setBusy(name);
        setErrorMessage(null);
        try {
            const res = await fetch(goApiUrl(`/api/worlds/${worldId}/agents`), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, role }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({ error: 'Unknown error' }));
                throw new Error(err.error || `deploy failed (${res.status})`);
            }
            onDeployed();
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'deploy failed');
        } finally {
            setBusy(null);
        }
    };

    const installAndDeploy = async (ex: EmptyAgentsExample) => {
        setBusy(ex.slug);
        setErrorMessage(null);
        try {
            // 1. Install the template (idempotent - keeps user edits).
            await apiPost(`/api/examples/${ex.slug}/install`);
            // 2. Deploy the example's primary agent into THIS world.
            const primary = ex.agents[0];
            if (!primary) {
                throw new Error('template has no agents');
            }
            await deploy(primary);
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'install failed');
            setBusy(null);
        }
    };

    const isLoading = installed === null || gallery === null;
    const noInstalled = installed && installed.length === 0;

    return (
        <div className="rounded-2xl border border-white/[0.06] bg-white/[0.01] px-5 py-6">
            <div className="mb-4 flex items-start gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-white/[0.08] bg-gradient-to-br from-blue-500/10 to-purple-500/10">
                    <IconUserPlus className="text-blue-400/80" size={16} />
                </div>
                <div className="min-w-0">
                    <h3 className="text-foreground/95 text-sm font-medium">This world is empty</h3>
                    <p className="text-muted-foreground/60 mt-0.5 text-[11px]">
                        Pick one of your agents to deploy here, or install a fresh one from the
                        gallery. Deployment is hot - no container restart.
                    </p>
                </div>
            </div>

            {errorMessage && <p className="mb-3 text-[11px] text-red-300/80">{errorMessage}</p>}

            {/* ── Already-installed agents ─────────────────────────────── */}
            {isLoading && (
                <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                    {[0, 1, 2].map((i) => (
                        <Skeleton className="h-16 rounded-lg" key={i} />
                    ))}
                </div>
            )}
            {!isLoading && noInstalled && (
                <div className="text-muted-foreground/60 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5 text-[11px]">
                    You don&apos;t have any agents installed yet. Pick one from the gallery below to
                    install and deploy in one click.
                </div>
            )}
            {!isLoading && !noInstalled && (
                <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                    {installed!.map((a) => (
                        <button
                            className="group flex items-center justify-between rounded-lg border border-white/[0.08] bg-white/[0.02] px-3 py-2.5 text-left transition-colors hover:border-white/[0.16] hover:bg-white/[0.05] disabled:cursor-not-allowed disabled:opacity-50"
                            disabled={busy !== null}
                            key={a.name}
                            onClick={async () => await deploy(a.name)}
                            type="button"
                        >
                            <div className="min-w-0">
                                <div className="text-foreground/90 truncate font-mono text-[12px]">
                                    {a.name}
                                </div>
                                <div className="text-muted-foreground/40 text-[10px] tracking-wider uppercase">
                                    worker
                                </div>
                            </div>
                            {busy === a.name ? (
                                <IconLoader2
                                    className="text-muted-foreground/60 shrink-0 animate-spin"
                                    size={13}
                                />
                            ) : (
                                <span className="text-muted-foreground/40 group-hover:text-foreground/80 inline-flex shrink-0 items-center gap-1 text-[10px] tracking-wider uppercase">
                                    Deploy
                                    <IconArrowRight size={11} />
                                </span>
                            )}
                        </button>
                    ))}
                </div>
            )}

            {/* ── Gallery toggle / panel ──────────────────────────────── */}
            <div className="mt-5">
                <button
                    className="text-muted-foreground/50 hover:text-foreground/80 inline-flex items-center gap-1.5 text-[11px] tracking-wider uppercase transition-colors"
                    onClick={() => setShowGallery((v) => !v)}
                    type="button"
                >
                    <IconSparkles size={11} />
                    {(() => {
                        if (showGallery) {
                            return 'Hide template gallery';
                        }
                        if (noInstalled) {
                            return 'Install one from a template';
                        }
                        return 'Or install a new one from a template';
                    })()}
                </button>

                {showGallery && (
                    <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                        {gallery && gallery.length > 0 ? (
                            gallery.map((ex) => (
                                <button
                                    className="group flex items-start gap-2.5 rounded-lg border border-white/[0.08] bg-white/[0.02] px-3 py-2.5 text-left transition-colors hover:border-white/[0.16] hover:bg-white/[0.05] disabled:cursor-not-allowed disabled:opacity-50"
                                    disabled={busy !== null}
                                    key={ex.slug}
                                    onClick={async () => await installAndDeploy(ex)}
                                    type="button"
                                >
                                    <IconWorld
                                        className="mt-0.5 shrink-0 text-blue-400/70"
                                        size={14}
                                    />
                                    <div className="min-w-0 flex-1">
                                        <div className="text-foreground/95 truncate text-[12px] font-medium">
                                            {ex.name}
                                        </div>
                                        <div className="text-muted-foreground/60 truncate text-[10px]">
                                            {ex.tagline}
                                        </div>
                                        <div className="mt-1 flex flex-wrap gap-1">
                                            {ex.agents.slice(0, 3).map((a) => (
                                                <span
                                                    className="text-muted-foreground/60 rounded border border-white/[0.06] bg-white/[0.03] px-1 py-0.5 font-mono text-[9px]"
                                                    key={a}
                                                >
                                                    {a}
                                                </span>
                                            ))}
                                        </div>
                                    </div>
                                    {busy === ex.slug && (
                                        <IconLoader2
                                            className="text-muted-foreground/60 shrink-0 animate-spin"
                                            size={12}
                                        />
                                    )}
                                </button>
                            ))
                        ) : (
                            <p className="text-muted-foreground/60 col-span-full text-[11px]">
                                No examples bundled in this build.
                            </p>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
}
