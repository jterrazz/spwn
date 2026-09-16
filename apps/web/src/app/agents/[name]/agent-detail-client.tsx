'use client';

import {
    IconBook,
    IconBrain,
    IconChevronDown,
    IconChevronRight,
    IconDownload,
    IconFile,
    IconFolder,
    IconFolderOpen,
    IconGitFork,
    IconMessageCircle,
    IconNotebook,
    IconPlanet,
    IconRefresh,
    IconRocket,
    IconTerminal,
    IconTrash,
    IconUser,
    IconX,
} from '@tabler/icons-react';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useState } from 'react';

import { apiAction, apiDelete, apiGet, apiPut, encPath, goApiUrl } from '@/api/client';
import { streamChat } from '@/api/stream-chat';
import { ActionButton } from '@/components/action-button';
import { useRefetch } from '@/components/app-shell';
import { Chat, ChatSuggestions } from '@/components/chat';
import type { ChatBubble } from '@/components/chat';
import {
    ItemList,
    KeyValue,
    MetricGrid,
    SectionHeader,
    SectionLabel,
    Separator,
    StatusDot,
    SubLabel,
} from '@/components/ds';
import { InlineEdit, InlineTagsEdit } from '@/components/inline-edit';
import { PageHeader } from '@/components/page-header';
import { ProgressShimmer } from '@/components/progress-shimmer';
import { Skeleton } from '@/components/ui/skeleton';
import { AVAILABLE_ROLES, getWorldName } from '@/domain/model';
import type { AgentProfile, Organization, Team, World } from '@/domain/model';
import { usePageTitle } from '@/hooks/use-page-title';
import { useProgressMessages } from '@/hooks/use-progress-messages';
import { ROLE_BADGE } from '@/styles/status-colors';

export default function AgentProfilePageWrapper() {
    return (
        <Suspense>
            <AgentProfilePage />
        </Suspense>
    );
}

function AgentProfilePage() {
    const params = useParams();
    const router = useRouter();
    const searchParams = useSearchParams();
    const agentName = decodeURIComponent(params.name as string);
    const worldId = searchParams.get('world') ?? undefined;

    const [profile, setProfile] = useState<AgentProfile | null>(null);
    const [mindTree, setMindTree] = useState<Record<string, string[]>>({});
    const [loading, setLoading] = useState(true);
    const [actionLoading, setActionLoading] = useState<null | string>(null);
    const [feedback, setFeedback] = useState<null | string>(null);
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [deleting, setDeleting] = useState(false);
    const [activeTab, setActiveTab] = useState<'chat' | 'files' | 'profile'>('chat');
    const [showWizard, setShowWizard] = useState(false);
    const [showDeployDialog, setShowDeployDialog] = useState(false);
    const [availableTeams, setAvailableTeams] = useState<Team[]>([]);
    const [availableWorlds, setAvailableWorlds] = useState<World[]>([]);
    const [deployTargetWorld, setDeployTargetWorld] = useState('');
    const [deploying, setDeploying] = useState(false);
    const [deployError, setDeployError] = useState('');
    const [deployRole, setDeployRole] = useState('worker');
    const [organizations, setOrganizations] = useState<Organization[]>([]);
    const [worldData, setWorldData] = useState<null | World>(null);
    const refetchSidebar = useRefetch();

    const deployProgressMessage = useProgressMessages(deploying, [
        { after: 0, text: 'Deploying agent...' },
        { after: 5, text: 'Building Docker image (first run could take a few minutes)...' },
        { after: 30, text: 'Still building... installing dependencies...' },
        { after: 60, text: 'Almost there...' },
    ]);

    usePageTitle(agentName, 'Agent');

    const fetchProfile = useCallback(() => {
        Promise.all([
            apiGet<AgentProfile>(`/api/agents/${encPath(agentName)}`).catch(() => null),
            apiGet<Record<string, string[]>>(`/api/agents/${agentName}/mind`).catch(() => null),
        ]).then(([agentProfile, tree]) => {
            setProfile(agentProfile ?? null);
            setMindTree(tree ?? {});
            setLoading(false);
            // Show wizard for new agents without a purpose
            if (agentProfile && !agentProfile.purpose) {
                setShowWizard(true);
            }
        });
    }, [agentName]);

    useEffect(() => {
        fetchProfile();
        apiGet<Team[]>('/api/teams')
            .then((t) => setAvailableTeams(t ?? []))
            .catch(() => {});
        apiGet<World[]>('/api/worlds')
            .then((w) => setAvailableWorlds(w ?? []))
            .catch(() => {});
        apiGet<Organization[]>('/api/organizations')
            .then((h) => setOrganizations(h ?? []))
            .catch(() => {});
        if (worldId) {
            apiGet<World>(`/api/worlds/${worldId}`)
                .then((w) => setWorldData(w ?? null))
                .catch(() => {});
        }
    }, [fetchProfile, worldId]);

    // Auto-detect world from available worlds - works even when single-world fetch fails
    useEffect(() => {
        if (availableWorlds.length === 0) {
            return;
        }
        const match = availableWorlds.find(
            (candidate) =>
                candidate.agent === agentName || candidate.agents.some((a) => a.name === agentName),
        );
        if (match) {
            setWorldData(match);
        }
    }, [availableWorlds, agentName]);

    const showFeedback = (msg: string) => {
        setFeedback(msg);
        setTimeout(() => setFeedback(null), 2500);
    };

    const saveIdentityField = async (field: string, content: string): Promise<boolean> => {
        try {
            await apiPut(`/api/agents/${agentName}/identity`, { field, content });
            showFeedback(`${field} updated`);
            fetchProfile(); // Refresh data
            return true;
        } catch {
            showFeedback(`Error: failed to update ${field}`);
            return false;
        }
    };

    const callAction = async (action: string, body?: object): Promise<boolean> => {
        setActionLoading(action);
        try {
            const result = await apiAction(`/api/agents/${agentName}/${action}`, body);
            if (!result.ok) {
                showFeedback(`Error: ${result.error || 'Unknown error'}`);
                return false;
            }
            return true;
        } catch {
            showFeedback('Error: Failed to connect to API');
            return false;
        } finally {
            setActionLoading(null);
        }
    };

    const handleDelete = async () => {
        setDeleting(true);
        try {
            await apiDelete(`/api/agents/${encPath(agentName)}`);
            router.push('/');
        } catch {
            showFeedback('Error: Failed to delete agent');
            setDeleting(false);
            setShowDeleteConfirm(false);
        }
    };

    const handleDeploy = async () => {
        if (!deployTargetWorld) {
            return;
        }
        setDeploying(true);
        setDeployError('');
        try {
            const res = await fetch(goApiUrl(`/api/worlds/${deployTargetWorld}/agents`), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: agentName, role: deployRole }),
                signal: AbortSignal.timeout(600_000), // 10 min - first run may build Docker images
            });
            const data = await res.json().catch(() => ({}));
            if (!res.ok) {
                setDeployError(data.error || `Deploy failed (HTTP ${res.status})`);
                setDeploying(false);
                return;
            }
            refetchSidebar();
            setShowDeployDialog(false);
            router.push(`/agents/${encPath(agentName)}?world=${deployTargetWorld}`);
        } catch (error) {
            const msg = error instanceof Error ? error.message : 'Unknown error';
            setDeployError(`Failed to connect: ${msg}`);
            setDeploying(false);
        }
    };

    if (loading) {
        return (
            <div className="max-w-3xl space-y-6 p-8">
                <div className="flex items-center gap-4">
                    <Skeleton className="h-12 w-12 rounded-xl" />
                    <div className="space-y-2">
                        <Skeleton className="h-6 w-32" />
                        <Skeleton className="h-3 w-48" />
                    </div>
                </div>
                <Skeleton className="h-24 w-full rounded-xl" />
                <Skeleton className="h-16 w-full rounded-xl" />
                <div className="flex gap-2">
                    {[1, 2, 3, 4].map((i) => (
                        <Skeleton className="h-8 w-24 rounded-full" key={i} />
                    ))}
                </div>
                <Skeleton className="h-32 w-full rounded-xl" />
            </div>
        );
    }

    if (!profile) {
        return (
            <div className="flex min-h-[60vh] flex-col items-center justify-center p-8">
                <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl border border-white/[0.06] bg-white/[0.03]">
                    <IconUser className="text-muted-foreground/20" size={28} />
                </div>
                <p className="text-muted-foreground/50 font-heading text-lg">
                    Agent &quot;{agentName}&quot; not found
                </p>
                <p className="text-muted-foreground/30 mt-2 font-mono text-xs">
                    Create this agent with: spwn agent create {agentName}
                </p>
                <button
                    className="text-foreground/60 hover:text-foreground/80 mt-6 flex items-center gap-2 rounded-xl border border-white/[0.06] bg-white/[0.04] px-4 py-2.5 text-sm transition-all hover:bg-white/[0.08]"
                    onClick={() => {
                        apiAction('/api/agents', { name: agentName }).then((result) => {
                            if (result.ok) {
                                fetchProfile();
                                showFeedback('Agent created!');
                            } else {
                                showFeedback(`Error: ${result.error}`);
                            }
                        });
                    }}
                >
                    <IconRocket size={16} />
                    Create &quot;{agentName}&quot;
                </button>
            </div>
        );
    }

    const totalFiles = Object.values(mindTree).reduce((n, f) => n + (f?.length ?? 0), 0);

    const deployedAgent = worldData?.agents.find((a) => a.name === agentName);
    const worldName = worldData ? getWorldName(worldData) : undefined;

    const mainContent = (
        <div className="min-w-0 flex-1 space-y-6 p-4 md:space-y-8 md:p-8">
            <PageHeader
                actions={
                    <>
                        <ActionButton
                            compact
                            disabled={actionLoading !== null}
                            icon={<IconRefresh size={16} stroke={2.2} />}
                            label="Dream"
                            onClick={async () => {
                                if ((profile?.journal?.length ?? 0) === 0) {
                                    showFeedback(
                                        'Nothing to dream about yet - spawn the agent in a world first',
                                    );
                                    return;
                                }
                                const ok = await callAction('dream');
                                if (ok) {
                                    showFeedback(
                                        'Dream cycle complete - check playbooks for promoted patterns',
                                    );
                                    fetchProfile();
                                }
                            }}
                        />
                        <ActionButton
                            compact
                            disabled={actionLoading !== null}
                            icon={<IconGitFork size={16} stroke={2.2} />}
                            label="Fork"
                            onClick={async () => {
                                const target = prompt('Fork target name:');
                                if (!target) {
                                    return;
                                }
                                const ok = await callAction('fork', { target });
                                if (ok) {
                                    showFeedback(`Forked to "${target}"`);
                                }
                            }}
                        />
                        <ActionButton
                            compact
                            disabled={actionLoading !== null}
                            icon={<IconDownload size={16} stroke={2.2} />}
                            label="Export"
                            onClick={async () => {
                                const ok = await callAction('export');
                                if (ok) {
                                    showFeedback('Export complete!');
                                }
                            }}
                        />
                        <ActionButton
                            compact
                            disabled={actionLoading !== null || deploying}
                            icon={<IconPlanet size={16} stroke={2.2} />}
                            label="Deploy"
                            onClick={() => {
                                setDeployError('');
                                setShowDeployDialog(true);
                            }}
                        />
                        <ActionButton
                            compact
                            danger
                            disabled={actionLoading !== null || deleting}
                            icon={<IconTrash size={16} stroke={2.2} />}
                            label="Delete"
                            onClick={() => setShowDeleteConfirm(true)}
                        />
                    </>
                }
                description={`${worldData?.runtime ?? profile.engine} · ${profile.role}`}
                title={agentName}
            />

            {/* Deploy dialog - select a running world to deploy this agent into */}
            {showDeployDialog && (
                <div className="fixed inset-0 z-50 flex items-center justify-center">
                    <div
                        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                        onClick={() => !deploying && setShowDeployDialog(false)}
                    />
                    <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-md overflow-hidden rounded-2xl border border-white/[0.08] shadow-2xl backdrop-blur-md">
                        {/* Top shimmer bar */}
                        {deploying && (
                            <div className="h-0.5 w-full overflow-hidden bg-white/[0.04]">
                                <div
                                    className="h-full w-1/3 rounded-full bg-emerald-500/30"
                                    style={{ animation: 'progressSlide 1.5s ease-in-out infinite' }}
                                />
                            </div>
                        )}
                        <div className="p-6">
                            <h3 className="font-heading text-foreground/90 mb-1 text-lg">
                                Deploy to World
                            </h3>
                            <p className="text-muted-foreground/50 mb-5 text-sm">
                                Add{' '}
                                <span className="text-foreground/70 font-mono">{agentName}</span> to
                                a running world.
                            </p>
                            <label className="text-muted-foreground/40 mb-2 block text-[10px] tracking-[0.15em] uppercase">
                                Select World
                            </label>
                            {availableWorlds.length === 0 ? (
                                <p className="text-muted-foreground/40 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-2.5 text-[11px]">
                                    No running worlds. Spawn one first from the Worlds page.
                                </p>
                            ) : (
                                <div className="max-h-48 overflow-y-auto rounded-lg border border-white/[0.08] bg-white/[0.02]">
                                    {availableWorlds.map((w) => {
                                        const wName = getWorldName(w);
                                        const isSelected = deployTargetWorld === w.id;
                                        const alreadyDeployed = w.agents.some(
                                            (a) => a.name === agentName,
                                        );
                                        let stateClass: string;
                                        if (alreadyDeployed) {
                                            stateClass = 'opacity-40 cursor-not-allowed';
                                        } else if (isSelected) {
                                            stateClass = 'bg-white/[0.06]';
                                        } else {
                                            stateClass = 'hover:bg-white/[0.03]';
                                        }
                                        return (
                                            <button
                                                className={`flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors ${stateClass}`}
                                                disabled={alreadyDeployed}
                                                key={w.id}
                                                onClick={() =>
                                                    !alreadyDeployed && setDeployTargetWorld(w.id)
                                                }
                                            >
                                                <span
                                                    className={`h-2 w-2 shrink-0 rounded-full ${
                                                        isSelected
                                                            ? 'bg-emerald-400'
                                                            : 'bg-white/[0.15]'
                                                    }`}
                                                />
                                                <span className="min-w-0 flex-1">
                                                    <span className="text-foreground/80 block truncate text-sm">
                                                        {wName}
                                                    </span>
                                                    <span className="text-muted-foreground/35 font-mono text-[10px]">
                                                        {w.agents.length} agent
                                                        {w.agents.length === 1 ? '' : 's'}
                                                        {alreadyDeployed && ' · already deployed'}
                                                    </span>
                                                </span>
                                            </button>
                                        );
                                    })}
                                </div>
                            )}
                            {/* Role selector */}
                            <label className="text-muted-foreground/40 mt-4 mb-2 block text-[10px] tracking-[0.15em] uppercase">
                                Role
                            </label>
                            <select
                                className="text-foreground/80 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none disabled:opacity-50"
                                disabled={deploying}
                                onChange={(e) => setDeployRole(e.target.value)}
                                value={deployRole}
                            >
                                {(() => {
                                    // Try to get roles from the selected world's organization, fall back to AVAILABLE_ROLES
                                    const selectedWorld = availableWorlds.find(
                                        (w) => w.id === deployTargetWorld,
                                    );
                                    const worldOrganizationSlug = (
                                        selectedWorld as unknown as
                                            | Record<string, unknown>
                                            | undefined
                                    )?.organization as string | undefined;
                                    const org = worldOrganizationSlug
                                        ? organizations.find(
                                              (h) => h.slug === worldOrganizationSlug,
                                          )
                                        : null;
                                    const roleNames = org
                                        ? org.roles.map((r) => r.name)
                                        : [...AVAILABLE_ROLES];
                                    return roleNames.map((r) => (
                                        <option key={r} value={r}>
                                            {r}
                                        </option>
                                    ));
                                })()}
                            </select>
                            {deployError && (
                                <p className="mt-3 text-xs text-red-400/80">{deployError}</p>
                            )}
                            <div className="mt-6 flex justify-end gap-3">
                                <button
                                    className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-50"
                                    disabled={deploying}
                                    onClick={() => setShowDeployDialog(false)}
                                >
                                    Cancel
                                </button>
                                <button
                                    className="flex items-center gap-2 rounded-lg border border-emerald-500/20 bg-emerald-500/20 px-4 py-2 text-sm text-emerald-300 transition-colors hover:bg-emerald-500/30 disabled:opacity-50"
                                    disabled={deploying || !deployTargetWorld}
                                    onClick={handleDeploy}
                                >
                                    {deploying ? (
                                        <>
                                            <div className="h-3 w-3 animate-spin rounded-full border-2 border-emerald-300/40 border-t-emerald-300" />
                                            Deploying…
                                        </>
                                    ) : (
                                        <>
                                            <IconPlanet size={14} />
                                            Deploy
                                        </>
                                    )}
                                </button>
                            </div>
                            <ProgressShimmer active={deploying} message={deployProgressMessage} />
                        </div>
                    </div>
                </div>
            )}

            {/* Delete confirmation dialog */}
            {showDeleteConfirm && (
                <div className="fixed inset-0 z-50 flex items-center justify-center">
                    <div
                        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                        onClick={() => setShowDeleteConfirm(false)}
                    />
                    <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-sm rounded-2xl border border-white/[0.08] p-6 shadow-2xl backdrop-blur-md">
                        <h3 className="font-heading text-foreground/90 mb-2 text-lg">
                            Delete Agent
                        </h3>
                        <p className="text-muted-foreground/50 mb-6 text-sm">
                            Are you sure you want to delete{' '}
                            <span className="text-foreground/70 font-mono">{agentName}</span>? This
                            will permanently remove all mind files, memories, and identity data.
                        </p>
                        <div className="flex justify-end gap-3">
                            <button
                                className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04]"
                                onClick={() => setShowDeleteConfirm(false)}
                            >
                                Cancel
                            </button>
                            <button
                                className="rounded-lg border border-red-500/20 bg-red-500/20 px-4 py-2 text-sm text-red-400 transition-colors hover:bg-red-500/30 disabled:opacity-50"
                                disabled={deleting}
                                onClick={handleDelete}
                            >
                                {deleting ? (
                                    <span className="flex items-center gap-2">
                                        <span className="h-3 w-3 animate-spin rounded-full border-2 border-red-400/40 border-t-red-400" />
                                        Deleting...
                                    </span>
                                ) : (
                                    'Delete Agent'
                                )}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Get Started Wizard Banner */}
            {showWizard && profile && !profile.purpose && (
                <div className="animate-in fade-in slide-in-from-top-2 rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-5 duration-300">
                    <div className="mb-3 flex items-start justify-between">
                        <div>
                            <h3 className="font-heading text-sm text-emerald-300">
                                Set up {agentName}&apos;s identity
                            </h3>
                            <p className="mt-1 text-[11px] text-emerald-300/50">
                                Give this agent a purpose so it knows what to focus on.
                            </p>
                        </div>
                        <button
                            className="text-emerald-300/30 transition-colors hover:text-emerald-300/60"
                            onClick={() => setShowWizard(false)}
                        >
                            <IconX size={16} />
                        </button>
                    </div>
                    <div className="space-y-3">
                        <div>
                            <label className="mb-1.5 block text-[10px] tracking-widest text-emerald-300/40 uppercase">
                                Purpose
                            </label>
                            <input
                                className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-emerald-500/15 bg-white/[0.03] px-3 py-2.5 text-sm transition-colors focus:border-emerald-500/30 focus:outline-none"
                                onKeyDown={async (e) => {
                                    if (e.key === 'Enter') {
                                        const val = (e.target as HTMLInputElement).value.trim();
                                        if (val) {
                                            await saveIdentityField('purpose', val);
                                            setShowWizard(false);
                                        }
                                    }
                                }}
                                placeholder="What is this agent's purpose?"
                            />
                        </div>
                        <div className="grid grid-cols-2 gap-3">
                            <div>
                                <label className="mb-1.5 block text-[10px] tracking-widest text-emerald-300/40 uppercase">
                                    Profile
                                </label>
                                <input
                                    className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-emerald-500/15 bg-white/[0.03] px-3 py-2 text-sm transition-colors focus:border-emerald-500/30 focus:outline-none"
                                    onKeyDown={async (e) => {
                                        if (e.key === 'Enter') {
                                            const val = (e.target as HTMLInputElement).value.trim();
                                            if (val) {
                                                await saveIdentityField('profile', val);
                                            }
                                        }
                                    }}
                                    placeholder="Describe their profile..."
                                />
                            </div>
                            <div>
                                <label className="mb-1.5 block text-[10px] tracking-widest text-emerald-300/40 uppercase">
                                    Traits
                                </label>
                                <input
                                    className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-emerald-500/15 bg-white/[0.03] px-3 py-2 text-sm transition-colors focus:border-emerald-500/30 focus:outline-none"
                                    onKeyDown={async (e) => {
                                        if (e.key === 'Enter') {
                                            const val = (e.target as HTMLInputElement).value.trim();
                                            if (val) {
                                                const traits = val
                                                    .split(',')
                                                    .map((t) => `- ${t.trim()}`)
                                                    .join('\n');
                                                await saveIdentityField('traits', traits);
                                            }
                                        }
                                    }}
                                    placeholder="e.g. curious, creative, diligent"
                                />
                            </div>
                        </div>
                        <p className="font-mono text-[10px] text-emerald-300/30">
                            Press Enter in any field to save. Fill purpose to dismiss this banner.
                        </p>
                    </div>
                </div>
            )}

            {/* Tab switcher */}
            <div className="flex gap-1 border-b border-white/[0.06] pb-px">
                <button
                    className={`-mb-px flex items-center gap-1.5 border-b-2 px-4 py-2 text-xs font-medium transition-colors ${
                        activeTab === 'chat'
                            ? 'border-foreground/50 text-foreground/80'
                            : 'text-muted-foreground/40 hover:text-muted-foreground/60 border-transparent'
                    }`}
                    onClick={() => setActiveTab('chat')}
                >
                    <IconMessageCircle size={13} />
                    Chat
                </button>
                <button
                    className={`-mb-px border-b-2 px-4 py-2 text-xs font-medium transition-colors ${
                        activeTab === 'profile'
                            ? 'border-foreground/50 text-foreground/80'
                            : 'text-muted-foreground/40 hover:text-muted-foreground/60 border-transparent'
                    }`}
                    onClick={() => setActiveTab('profile')}
                >
                    Profile
                </button>
                <button
                    className={`-mb-px border-b-2 px-4 py-2 text-xs font-medium transition-colors ${
                        activeTab === 'files'
                            ? 'border-foreground/50 text-foreground/80'
                            : 'text-muted-foreground/40 hover:text-muted-foreground/60 border-transparent'
                    }`}
                    onClick={() => setActiveTab('files')}
                >
                    Files ({totalFiles})
                </button>
            </div>

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

            {/* Profile tab - diagnostics panel style */}
            {activeTab === 'profile' && (
                <>
                    <MetricGrid
                        className="gap-x-8 gap-y-4"
                        columns={2}
                        items={[
                            { label: 'Files', value: totalFiles },
                            { label: 'Journal', value: profile.journal?.length ?? 0 },
                            { label: 'Skills', value: profile.skills?.length ?? 0 },
                            { label: 'Traits', value: profile.traits?.length ?? 0 },
                        ]}
                    />

                    <Separator />

                    {/* Team */}
                    <div className="flex items-center justify-between">
                        <SubLabel>Team</SubLabel>
                        <select
                            className="text-foreground/80 cursor-pointer bg-transparent text-right font-mono text-sm focus:outline-none"
                            onChange={async (e) => {
                                const ok = await saveIdentityField('team', e.target.value);
                                if (ok) {
                                    fetchProfile();
                                }
                            }}
                            value={profile.team ?? ''}
                        >
                            <option value="">-</option>
                            {availableTeams.map((t) => (
                                <option key={t.slug} value={t.slug}>
                                    {t.name}
                                </option>
                            ))}
                        </select>
                    </div>

                    {/* Deployment History */}
                    {(() => {
                        const journalFiles = mindTree.journal ?? [];
                        const worldIds = journalFiles
                            .filter((f) => f.endsWith('.json'))
                            .map((f) => f.replace('.json', ''));
                        if (worldIds.length === 0) {
                            return null;
                        }
                        return (
                            <>
                                <Separator />
                                <div>
                                    <SectionHeader>Deployment History</SectionHeader>
                                    <ItemList
                                        items={worldIds.map((wid) => {
                                            const parts = wid.split('-');
                                            const second = parts[1];
                                            const wName =
                                                second === undefined
                                                    ? wid
                                                    : second.charAt(0).toUpperCase() +
                                                      second.slice(1);
                                            return {
                                                name: wName,
                                                detail: wid,
                                                href: `/agents/${encPath(agentName)}?world=${wid}`,
                                            };
                                        })}
                                    />
                                </div>
                            </>
                        );
                    })()}

                    <Separator />

                    {/* Identity */}
                    <div>
                        <SectionHeader>Identity</SectionHeader>

                        <div className="mb-4">
                            <SubLabel className="mb-1.5">Purpose</SubLabel>
                            <InlineEdit
                                className="text-foreground/75 text-sm leading-relaxed"
                                multiline
                                onSave={async (v) => await saveIdentityField('purpose', v)}
                                placeholder="Define this agent's purpose..."
                                value={profile.purpose || ''}
                            />
                        </div>

                        <div className="mb-4">
                            <SubLabel className="mb-1.5">Profile</SubLabel>
                            <InlineEdit
                                className="text-foreground/60 text-sm leading-relaxed italic"
                                multiline
                                onSave={async (v) => await saveIdentityField('profile', v)}
                                placeholder="Describe the agent's profile..."
                                value={profile.profile || ''}
                            />
                        </div>

                        <div>
                            <SubLabel className="mb-1.5">Traits</SubLabel>
                            <InlineTagsEdit
                                color="bg-white/[0.06] text-foreground/60 border-white/[0.08]"
                                onSave={async (tags) => {
                                    const ok = await saveIdentityField(
                                        'traits',
                                        tags.map((t) => `- ${t}`).join('\n'),
                                    );
                                    return ok;
                                }}
                                tags={profile.traits ?? []}
                            />
                        </div>
                    </div>

                    {/* Skills */}
                    {(profile.skills?.length ?? 0) > 0 && (
                        <>
                            <Separator />
                            <div>
                                <SectionHeader>Skills</SectionHeader>
                                <div className="flex flex-wrap gap-1.5">
                                    {(profile.skills ?? []).map((skill) => (
                                        <span
                                            className="text-foreground/60 border border-white/[0.06] bg-white/[0.04] px-2.5 py-1 font-mono text-[11px]"
                                            key={skill}
                                        >
                                            {skill}
                                        </span>
                                    ))}
                                </div>
                            </div>
                        </>
                    )}

                    {/* Journal */}
                    {(profile.journal?.length ?? 0) > 0 && (
                        <>
                            <Separator />
                            <div>
                                <SectionHeader>Journal</SectionHeader>
                                <div className="space-y-3">
                                    {(profile.journal ?? []).map((entry) => (
                                        <div key={entry.date}>
                                            <SubLabel className="mb-1">{entry.date}</SubLabel>
                                            <p className="text-foreground/60 text-xs leading-relaxed">
                                                {entry.summary}
                                            </p>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </>
                    )}
                </>
            )}

            {/* Chat tab */}
            {activeTab === 'chat' && <AgentChat agentName={agentName} worldId={worldId} />}

            {/* Files tab */}
            {activeTab === 'files' && <MindFileViewer agentName={agentName} mindTree={mindTree} />}
        </div>
    );

    return (
        <div className="flex h-[calc(100vh-1px)] overflow-hidden">
            <div className="min-w-0 flex-1 overflow-y-auto">{mainContent}</div>
            {/* ── Right: Quick info panel ── */}
            <div className="border-border/30 hidden w-72 shrink-0 overflow-y-auto border-l lg:block">
                <div className="space-y-6 p-5">
                    <SectionHeader>Diagnostics</SectionHeader>

                    {/* Identity */}
                    <div>
                        <SectionLabel>Identity</SectionLabel>
                        <div className="space-y-2">
                            <KeyValue label="Name" value={agentName} />
                            <div className="flex items-center justify-between">
                                <SubLabel>Role</SubLabel>
                                {(() => {
                                    const role = deployedAgent?.role ?? profile.role;
                                    const badge = ROLE_BADGE[role] ?? ROLE_BADGE.default;
                                    return (
                                        <span
                                            className={`rounded border px-1.5 py-0.5 font-mono text-[9px] tracking-wider uppercase ${badge}`}
                                        >
                                            {role}
                                        </span>
                                    );
                                })()}
                            </div>
                            {deployedAgent && (
                                <div className="flex items-center justify-between">
                                    <SubLabel>Status</SubLabel>
                                    <div className="flex items-center gap-1.5">
                                        <StatusDot status={deployedAgent.status} />
                                        <span className="text-foreground/80 font-mono text-xs font-medium capitalize">
                                            {deployedAgent.status}
                                        </span>
                                    </div>
                                </div>
                            )}
                            <KeyValue
                                label="Runtime"
                                value={worldData?.runtime ?? profile.engine}
                            />
                            {profile.team && <KeyValue label="Team" value={profile.team} />}
                        </div>
                    </div>

                    <Separator />

                    {/* Stats */}
                    <div>
                        <SectionLabel>Metrics</SectionLabel>
                        <MetricGrid
                            columns={2}
                            items={[
                                { label: 'Files', value: totalFiles },
                                { label: 'Journal', value: profile.journal?.length ?? 0 },
                                { label: 'Skills', value: profile.skills?.length ?? 0 },
                                { label: 'Traits', value: profile.traits?.length ?? 0 },
                            ]}
                        />
                    </div>

                    {/* World info - only when deployed */}
                    {worldData && (
                        <>
                            <Separator />
                            <div>
                                <SectionLabel>World</SectionLabel>
                                <div className="space-y-2">
                                    <KeyValue label="Name" value={worldName ?? worldId!} />
                                    <div className="flex items-center justify-between">
                                        <SubLabel>Status</SubLabel>
                                        <div className="flex items-center gap-1.5">
                                            <StatusDot status={worldData.status} />
                                            <span className="text-foreground/80 font-mono text-xs font-medium capitalize">
                                                {worldData.status}
                                            </span>
                                        </div>
                                    </div>
                                    <KeyValue label="Agents" value={worldData.agents.length} />
                                    {worldData.workspaces && worldData.workspaces.length > 0 && (
                                        <KeyValue
                                            label="Workspaces"
                                            value={worldData.workspaces.length}
                                        />
                                    )}
                                </div>
                            </div>
                        </>
                    )}

                    <Separator />
                </div>
            </div>
        </div>
    );
}

/* ── Agent Chat ── */

function AgentChat({ agentName, worldId }: { agentName: string; worldId?: string | undefined }) {
    const [messages, setMessages] = useState<ChatBubble[]>([]);
    const [sending, setSending] = useState(false);

    // Load conversation history when in world context
    useEffect(() => {
        if (!worldId) {
            return;
        }
        apiGet<{
            sessions: {
                messages: {
                    role: string;
                    content: string;
                    timestamp: string;
                    type: string;
                    toolName?: string;
                    cost?: number;
                    durationMs?: number;
                }[];
            }[];
        }>(`/api/worlds/${worldId}/history?agent=${encPath(agentName)}`)
            .then((data) => {
                if (!data?.sessions?.length) {
                    return;
                }
                const historyMsgs: ChatBubble[] = [];
                for (const session of data.sessions) {
                    for (const msg of session.messages) {
                        if (msg.type === 'text' && msg.content) {
                            historyMsgs.push({
                                role: msg.role === 'user' ? 'user' : 'assistant',
                                content: msg.content,
                                blocks: [{ type: 'text', content: msg.content }],
                                timestamp: new Date(msg.timestamp),
                            });
                        }
                    }
                }
                if (historyMsgs.length > 0) {
                    setMessages(historyMsgs);
                }
            })
            .catch(() => {});
    }, [worldId, agentName]);

    const handleSend = async (msg: string) => {
        setMessages((prev) => [
            ...prev,
            {
                role: 'user',
                content: msg,
                blocks: [{ type: 'text', content: msg }],
                timestamp: new Date(),
            },
        ]);
        setSending(true);

        const assistantIndex = messages.length + 1;
        setMessages((prev) => [
            ...prev,
            { role: 'assistant', content: '', blocks: [], timestamp: new Date() },
        ]);

        // When in world context, talk through the world endpoint (which routes to the deployed container).
        // Otherwise, use the standalone agent talk endpoint (limbo / direct).
        const chatUrl = worldId
            ? goApiUrl(`/api/worlds/${worldId}/talk`)
            : goApiUrl(`/api/agents/${encPath(agentName)}/talk`);
        const chatBody = worldId ? { message: msg, agent: agentName } : { message: msg };

        await streamChat({
            url: chatUrl,
            body: chatBody,
            onBlocks: (newBlocks) => {
                setMessages((prev) => {
                    const updated = [...prev];
                    const last = updated[assistantIndex];
                    if (last && last.role === 'assistant') {
                        const allBlocks = [...last.blocks, ...newBlocks];
                        const textContent = allBlocks
                            .filter(
                                (b): b is { type: 'text'; content: string } => b.type === 'text',
                            )
                            .map((b) => b.content)
                            .join('');
                        updated[assistantIndex] = {
                            ...last,
                            blocks: allBlocks,
                            content: textContent,
                        };
                    }
                    return updated;
                });
            },
            onDone: (meta) => {
                setMessages((prev) => {
                    const updated = [...prev];
                    const last = updated[assistantIndex];
                    if (last && last.role === 'assistant') {
                        updated[assistantIndex] = {
                            ...last,
                            cost: meta.cost,
                            duration: meta.duration,
                        };
                    }
                    return updated;
                });
            },
            onError: (error) => {
                setMessages((prev) => {
                    const updated = [...prev];
                    const last = updated[assistantIndex];
                    if (last && last.role === 'assistant') {
                        updated[assistantIndex] = {
                            ...last,
                            content: error,
                            blocks: [{ type: 'error', content: error }],
                            error: true,
                        };
                    }
                    return updated;
                });
            },
        });

        setSending(false);
    };

    return (
        <Chat
            assistantLabel={agentName}
            autoFocus
            className="h-[calc(100vh-220px)] min-h-[360px]"
            disabled={sending}
            emptyState={
                <div className="flex flex-col items-center justify-center text-center">
                    <IconTerminal className="text-muted-foreground/15 mb-3" size={28} />
                    <p className="text-muted-foreground/30 text-sm">Chat with {agentName}</p>
                    <p className="text-muted-foreground/20 mt-1 mb-4 text-[11px]">
                        Send messages directly to this agent in real-time
                    </p>
                    <ChatSuggestions
                        onPick={async (s) => await handleSend(s)}
                        suggestions={[
                            'What are you working on?',
                            'Show me the project structure',
                            'Run the tests',
                        ]}
                    />
                </div>
            }
            messages={messages}
            onSend={handleSend}
            placeholder={`Message ${agentName}...`}
            typingText={`${agentName} is thinking…`}
        />
    );
}

/* ── Mind File Viewer ── */

const LAYER_ICONS: Record<string, typeof IconFolder> = {
    core: IconUser,
    skills: IconBrain,
    knowledge: IconBook,
    playbooks: IconBook,
    journal: IconNotebook,
};

function MindFileViewer({
    agentName,
    mindTree,
}: {
    agentName: string;
    mindTree: Record<string, string[]>;
}) {
    const [expandedLayers, setExpandedLayers] = useState<Set<string>>(new Set());
    const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());
    const [fileContents, setFileContents] = useState<Record<string, string>>({});
    const [loadingFiles, setLoadingFiles] = useState<Set<string>>(new Set());

    const toggleLayer = (layer: string) => {
        setExpandedLayers((prev) => {
            const next = new Set(prev);
            if (next.has(layer)) {
                next.delete(layer);
            } else {
                next.add(layer);
            }
            return next;
        });
    };

    const toggleFile = async (layer: string, file: string) => {
        const key = `${layer}/${file}`;
        const fullPath = file.endsWith('.md') ? `${layer}/${file}` : `${layer}/${file}.md`;

        if (expandedFiles.has(key)) {
            setExpandedFiles((prev) => {
                const next = new Set(prev);
                next.delete(key);
                return next;
            });
            return;
        }

        setExpandedFiles((prev) => new Set(prev).add(key));

        // Fetch content if not cached
        if (!fileContents[key]) {
            setLoadingFiles((prev) => new Set(prev).add(key));
            try {
                const res = await fetch(goApiUrl(`/api/agents/${agentName}/files/${fullPath}`));
                if (res.ok) {
                    const data = await res.json();
                    setFileContents((prev) => ({ ...prev, [key]: data.content }));
                } else {
                    setFileContents((prev) => ({
                        ...prev,
                        [key]: '⚠ Failed to load file content',
                    }));
                }
            } catch {
                setFileContents((prev) => ({ ...prev, [key]: '⚠ Failed to connect to API' }));
            } finally {
                setLoadingFiles((prev) => {
                    const next = new Set(prev);
                    next.delete(key);
                    return next;
                });
            }
        }
    };

    const sortedLayers = Object.keys(mindTree).sort((a, b) => {
        const order = ['identity', 'skills', 'playbooks', 'journal'];
        const rank = (key: string) => (order.includes(key) ? order.indexOf(key) : 99);
        return rank(a) - rank(b);
    });

    if (sortedLayers.length === 0) {
        return (
            <div className="py-12 text-center">
                <IconFolder className="text-muted-foreground/15 mx-auto mb-3" size={32} />
                <p className="text-muted-foreground/40 text-sm">No mind files found</p>
                <p className="text-muted-foreground/25 mt-1 font-mono text-xs">
                    Create files with: spwn agent dream {agentName}
                </p>
            </div>
        );
    }

    return (
        <div className="space-y-2">
            {sortedLayers.map((layer) => {
                const files = mindTree[layer] ?? [];
                const isExpanded = expandedLayers.has(layer);
                const LayerIcon = LAYER_ICONS[layer] ?? IconFolder;

                return (
                    <div className="glass-subtle overflow-hidden" key={layer}>
                        {/* Layer header */}
                        <button
                            className="flex w-full items-center gap-2.5 px-4 py-3 text-left transition-colors hover:bg-white/[0.02]"
                            onClick={() => toggleLayer(layer)}
                        >
                            {isExpanded ? (
                                <IconChevronDown
                                    className="text-muted-foreground/40 shrink-0"
                                    size={14}
                                />
                            ) : (
                                <IconChevronRight
                                    className="text-muted-foreground/40 shrink-0"
                                    size={14}
                                />
                            )}
                            {isExpanded ? (
                                <IconFolderOpen className="text-foreground/50 shrink-0" size={16} />
                            ) : (
                                <LayerIcon className="text-foreground/40 shrink-0" size={16} />
                            )}
                            <span className="text-foreground/70 flex-1 font-mono text-xs">
                                {layer}/
                            </span>
                            <span className="text-muted-foreground/30 font-mono text-[10px]">
                                {files.length} files
                            </span>
                        </button>

                        {/* Files list */}
                        {isExpanded && files.length > 0 && (
                            <div className="border-t border-white/[0.04]">
                                {files.map((file) => {
                                    const key = `${layer}/${file}`;
                                    const isFileExpanded = expandedFiles.has(key);
                                    const isLoading = loadingFiles.has(key);
                                    const content = fileContents[key];

                                    return (
                                        <div key={file}>
                                            <button
                                                className="flex w-full items-center gap-2.5 px-4 py-2 pl-10 text-left transition-colors hover:bg-white/[0.02]"
                                                onClick={async () => await toggleFile(layer, file)}
                                            >
                                                {isFileExpanded ? (
                                                    <IconChevronDown
                                                        className="text-muted-foreground/30 shrink-0"
                                                        size={12}
                                                    />
                                                ) : (
                                                    <IconChevronRight
                                                        className="text-muted-foreground/30 shrink-0"
                                                        size={12}
                                                    />
                                                )}
                                                <IconFile
                                                    className="text-muted-foreground/30 shrink-0"
                                                    size={13}
                                                />
                                                <span className="text-foreground/60 font-mono text-[11px]">
                                                    {file}
                                                </span>
                                            </button>

                                            {/* File content */}
                                            {isFileExpanded && (
                                                <div className="border-t border-white/[0.03] bg-white/[0.01] px-4 py-3 pl-16">
                                                    {isLoading ? (
                                                        <div className="text-muted-foreground/30 flex items-center gap-2 text-xs">
                                                            <div className="border-foreground/20 border-t-foreground/50 h-3 w-3 animate-spin rounded-full border-2" />
                                                            Loading...
                                                        </div>
                                                    ) : (
                                                        <pre className="text-foreground/50 max-h-96 overflow-x-auto overflow-y-auto font-mono text-[11px] leading-relaxed whitespace-pre-wrap">
                                                            {content ?? 'No content'}
                                                        </pre>
                                                    )}
                                                </div>
                                            )}
                                        </div>
                                    );
                                })}
                            </div>
                        )}
                    </div>
                );
            })}
        </div>
    );
}
