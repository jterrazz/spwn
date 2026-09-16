'use client';

import { IconCheck, IconPlus, IconUser, IconUsers, IconX } from '@tabler/icons-react';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { apiAction, apiGet, goApiUrl } from '@/api/client';
import { ActionButton } from '@/components/action-button';
import { useRefetch } from '@/components/app-shell';
import { DataTable, SectionLabel, StatusDot } from '@/components/ds';
import { ExpandingSearch } from '@/components/expanding-search';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { Skeleton } from '@/components/ui/skeleton';
import { getWorldName } from '@/domain/model';
import type { Team, World } from '@/domain/model';
import { usePageTitle } from '@/hooks/use-page-title';
import { ROLE_BADGE } from '@/styles/status-colors';

interface AgentListItem {
    name: string;
    path: string;
    team?: string;
    role?: string;
    layers: Record<string, null | string[]>;
}

// An agent enriched with its current deployment (if any).
interface EnrichedAgent {
    name: string;
    role: string;
    team?: string | undefined; // Team slug
    status: string; // Running/waiting/idle/sleeping/stopped/limbo
    worldID?: string | undefined;
    worldName?: string | undefined;
    journalEntries: number;
    sessionsCount: number;
}

type StatusFilter = 'all' | 'deployed' | 'limbo';

// Hoisted out of the page component so cell renderers are stable across renders.
const AGENT_COLUMNS = [
    {
        key: 'name',
        label: 'Name',
        width: '1fr',
        render: (a: EnrichedAgent) => (
            <span className="text-foreground/85 truncate font-mono text-[13px]">{a.name}</span>
        ),
    },
    {
        key: 'role',
        label: 'Role',
        width: '80px',
        render: (a: EnrichedAgent) => {
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
        render: (a: EnrichedAgent) => (
            <span className="flex items-center gap-1.5">
                <StatusDot status={a.status === 'limbo' ? 'stopped' : a.status} />
                <span className="text-muted-foreground/50 font-mono text-[11px] capitalize">
                    {a.status}
                </span>
            </span>
        ),
    },
    {
        key: 'world',
        label: 'World',
        width: '120px',
        render: (a: EnrichedAgent) =>
            a.worldName ? (
                <span className="text-foreground/60 truncate font-mono text-[11px]">
                    {a.worldName}
                </span>
            ) : (
                <span className="text-muted-foreground/25 font-mono text-[11px]">-</span>
            ),
    },
];

function countLayerFiles(layers: Record<string, null | string[]>): {
    journal: number;
    sessions: number;
} {
    let journal = 0;
    let sessions = 0;
    for (const [key, val] of Object.entries(layers ?? {})) {
        if (!Array.isArray(val) || val.length === 0) {
            continue;
        }
        if (key === 'memory/journal') {
            journal = val.length;
        }
        if (key === 'sessions') {
            sessions = val.length;
        }
    }
    return { journal, sessions };
}

export default function AgentsPage() {
    const router = useRouter();
    const refetchSidebar = useRefetch();
    usePageTitle('Agents');

    const [agents, setAgents] = useState<EnrichedAgent[]>([]);
    const [teams, setTeams] = useState<Team[]>([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState<StatusFilter>('all');
    const [query, setQuery] = useState('');
    const [showNew, setShowNew] = useState(false);
    const [newName, setNewName] = useState('');
    const [newTeam, setNewTeam] = useState('');
    const [creating, setCreating] = useState(false);
    const [createError, setCreateError] = useState('');
    const [showTeamDialog, setShowTeamDialog] = useState(false);
    const [editingTeam, setEditingTeam] = useState<null | Team>(null);
    const [teamName, setTeamName] = useState('');
    const [teamColor, setTeamColor] = useState('');
    const [teamDesc, setTeamDesc] = useState('');
    const [savingTeam, setSavingTeam] = useState(false);

    const fetchAll = useCallback(async () => {
        try {
            const [worlds, rawAgents, rawTeams] = await Promise.all([
                apiGet<World[]>('/api/worlds').catch(() => [] as World[]),
                apiGet<AgentListItem[]>('/api/agents').catch(() => [] as AgentListItem[]),
                apiGet<Team[]>('/api/teams').catch(() => [] as Team[]),
            ]);

            setTeams(rawTeams ?? []);

            // Build name → { worldID, worldName, role, status } map from world records.
            const placement = new Map<
                string,
                { worldID: string; worldName: string; role: string; status: string }
            >();
            for (const w of worlds) {
                for (const a of w.agents ?? []) {
                    placement.set(a.name, {
                        worldID: w.id,
                        worldName: getWorldName(w),
                        role: a.role ?? 'worker',
                        status: a.status ?? 'idle',
                    });
                }
            }

            const enriched: EnrichedAgent[] = rawAgents.map((a) => {
                const counts = countLayerFiles(a.layers ?? {});
                const p = placement.get(a.name);
                return {
                    name: a.name,
                    role: p?.role ?? a.role ?? 'worker',
                    team: a.team,
                    status: p?.status ?? 'limbo',
                    worldID: p?.worldID,
                    worldName: p?.worldName,
                    journalEntries: counts.journal,
                    sessionsCount: counts.sessions,
                };
            });

            enriched.sort((a, b) => a.name.localeCompare(b.name));
            setAgents(enriched);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchAll();
        const id = setInterval(fetchAll, 5000);
        return () => clearInterval(id);
    }, [fetchAll]);

    const filtered = useMemo(() => {
        const q = query.trim().toLowerCase();
        return agents.filter((a) => {
            if (filter === 'deployed' && !a.worldID) {
                return false;
            }
            if (filter === 'limbo' && a.worldID) {
                return false;
            }
            if (q && !a.name.toLowerCase().includes(q)) {
                return false;
            }
            return true;
        });
    }, [agents, filter, query]);

    const counts = useMemo(
        () => ({
            all: agents.length,
            deployed: agents.filter((a) => a.worldID).length,
            limbo: agents.filter((a) => !a.worldID).length,
        }),
        [agents],
    );

    // Group filtered agents by team for the list view.
    const grouped = useMemo(() => {
        const teamMap = new Map<string, Team>();
        for (const t of teams) {
            teamMap.set(t.slug, t);
        }

        const groups: { team: null | Team; agents: EnrichedAgent[] }[] = [];
        const bySlug = new Map<string, EnrichedAgent[]>();
        const solo: EnrichedAgent[] = [];

        for (const a of filtered) {
            if (a.team) {
                const list = bySlug.get(a.team) ?? [];
                list.push(a);
                bySlug.set(a.team, list);
            } else {
                solo.push(a);
            }
        }

        // Teams first (sorted by name), then solo
        for (const [slug, members] of bySlug) {
            groups.push({ team: teamMap.get(slug) ?? { slug, name: slug }, agents: members });
        }
        groups.sort((a, b) => (a.team?.name ?? '').localeCompare(b.team?.name ?? ''));
        if (solo.length > 0) {
            groups.push({ team: null, agents: solo });
        }
        return groups;
    }, [filtered, teams]);

    const handleCreate = async () => {
        const name = newName.trim();
        if (!name) {
            return;
        }
        setCreating(true);
        setCreateError('');
        try {
            const result = await apiAction('/api/agents', { name });
            if (!result.ok) {
                setCreateError(result.error || 'Failed to create agent');
                setCreating(false);
                return;
            }
            // Assign team if selected
            if (newTeam) {
                await fetch(goApiUrl(`/api/agents/${encodeURIComponent(name)}/identity`), {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ field: 'team', content: newTeam }),
                }).catch(() => {});
            }
            setNewName('');
            setNewTeam('');
            setShowNew(false);
            refetchSidebar();
            await fetchAll();
            router.push(`/agents/${name}`);
        } catch {
            setCreateError('Failed to connect to API');
        } finally {
            setCreating(false);
        }
    };

    const openTeamDialog = (t?: Team) => {
        if (t) {
            setEditingTeam(t);
            setTeamName(t.name);
            setTeamColor(t.color ?? '');
            setTeamDesc(t.description ?? '');
        } else {
            setEditingTeam(null);
            setTeamName('');
            setTeamColor('');
            setTeamDesc('');
        }
        setShowTeamDialog(true);
    };

    const handleSaveTeam = async () => {
        if (!teamName.trim()) {
            return;
        }
        setSavingTeam(true);
        try {
            const body = {
                name: teamName.trim(),
                color: teamColor.trim(),
                description: teamDesc.trim(),
            };
            await fetch(
                editingTeam ? goApiUrl(`/api/teams/${editingTeam.slug}`) : goApiUrl('/api/teams'),
                {
                    method: editingTeam ? 'PUT' : 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(body),
                },
            );
            setShowTeamDialog(false);
            await fetchAll();
        } catch {
            // Ignore
        } finally {
            setSavingTeam(false);
        }
    };

    const handleDeleteTeam = async (slug: string) => {
        await fetch(goApiUrl(`/api/teams/${slug}`), { method: 'DELETE' }).catch(() => {});
        await fetchAll();
    };

    return (
        <Page>
            <PageHeader
                actions={
                    <>
                        <ExpandingSearch
                            onChange={setQuery}
                            placeholder="Search agents…"
                            value={query}
                        />
                        <ActionButton
                            compact
                            icon={<IconUsers size={16} stroke={2.2} />}
                            label="New Team"
                            onClick={() => openTeamDialog()}
                        />
                        <ActionButton
                            compact
                            icon={<IconPlus size={18} stroke={2.4} />}
                            label="New Agent"
                            onClick={() => {
                                setCreateError('');
                                setNewTeam('');
                                setShowNew(true);
                            }}
                        />
                    </>
                }
                description="Persistent identities that remember across sessions and worlds."
                title="Agents"
            />

            {/* Filter */}
            <div className="flex flex-wrap items-center gap-3">
                <div className="glass-pill flex items-center gap-1 px-1 py-1">
                    {(['all', 'deployed', 'limbo'] as StatusFilter[]).map((f) => (
                        <button
                            className={`rounded-full px-3 py-1 text-xs capitalize transition-colors ${
                                filter === f
                                    ? 'text-foreground/90 bg-white/[0.1]'
                                    : 'text-muted-foreground/50 hover:text-foreground/70'
                            }`}
                            key={f}
                            onClick={() => setFilter(f)}
                        >
                            {f}{' '}
                            <span className="text-muted-foreground/40 ml-1 font-mono text-[10px]">
                                {counts[f]}
                            </span>
                        </button>
                    ))}
                </div>
            </div>

            {/* List */}
            {loading && (
                <div className="space-y-2">
                    {[1, 2, 3].map((i) => (
                        <Skeleton className="h-14 w-full rounded-xl" key={i} />
                    ))}
                </div>
            )}
            {!loading && filtered.length === 0 && (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                    <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl border border-white/[0.06] bg-white/[0.03]">
                        <IconUser className="text-muted-foreground/30" size={24} />
                    </div>
                    <p className="text-muted-foreground/50 text-sm">
                        {query
                            ? 'No agents match your search'
                            : 'No agents yet. Create one to get started.'}
                    </p>
                </div>
            )}
            {!loading && filtered.length > 0 && (
                <div className="space-y-8">
                    {grouped.map(({ team: t, agents: groupAgents }) => (
                        <div key={t?.slug ?? 'solo'}>
                            {/* Team header */}
                            <div className="mb-3 flex items-center gap-2">
                                {t ? (
                                    <>
                                        <button
                                            className="underline-offset-2 transition-colors hover:underline"
                                            onClick={() => openTeamDialog(t)}
                                            style={t.color ? { color: t.color } : undefined}
                                        >
                                            <SectionLabel className="mb-0">{t.name}</SectionLabel>
                                        </button>
                                        <span className="text-muted-foreground/30 font-mono text-[10px]">
                                            {groupAgents.length}
                                        </span>
                                    </>
                                ) : (
                                    <>
                                        <SectionLabel className="text-muted-foreground/30 mb-0">
                                            No team
                                        </SectionLabel>
                                        <span className="text-muted-foreground/20 font-mono text-[10px]">
                                            {groupAgents.length}
                                        </span>
                                    </>
                                )}
                            </div>
                            {/* Agent table */}
                            <DataTable<EnrichedAgent>
                                columns={AGENT_COLUMNS}
                                rowHref={(a) =>
                                    a.worldID
                                        ? `/agents/${encodeURIComponent(a.name)}?world=${a.worldID}`
                                        : `/agents/${encodeURIComponent(a.name)}`
                                }
                                rowKey={(a) => a.name}
                                rows={groupAgents}
                            />
                        </div>
                    ))}
                </div>
            )}

            {/* New Agent dialog */}
            {showNew && (
                <div className="fixed inset-0 z-50 flex items-center justify-center">
                    <div
                        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                        onClick={() => !creating && setShowNew(false)}
                    />
                    <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-md rounded-2xl border border-white/[0.08] p-6 shadow-2xl backdrop-blur-md">
                        <h3 className="font-heading text-foreground/90 mb-1 text-lg">New Agent</h3>
                        <p className="text-muted-foreground/50 mb-5 text-sm">
                            Creates a new agent identity in limbo. Deploy it to a world when ready.
                        </p>
                        <input
                            autoFocus
                            className="text-foreground/80 placeholder:text-muted-foreground/30 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 font-mono text-sm transition-colors focus:border-white/[0.16] focus:outline-none disabled:opacity-50"
                            disabled={creating}
                            onChange={(e) => setNewName(e.target.value)}
                            onKeyDown={(e) => {
                                if (e.key === 'Enter') {
                                    handleCreate();
                                }
                                if (e.key === 'Escape') {
                                    setShowNew(false);
                                }
                            }}
                            placeholder="e.g. atlas, morpheus, neo…"
                            value={newName}
                        />
                        <label className="text-muted-foreground/40 mt-4 mb-1.5 block text-[10px] tracking-widest uppercase">
                            Team{' '}
                            <span className="text-muted-foreground/25 tracking-normal normal-case">
                                (optional)
                            </span>
                        </label>
                        <select
                            className="text-foreground/80 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none disabled:opacity-50"
                            disabled={creating}
                            onChange={(e) => setNewTeam(e.target.value)}
                            value={newTeam}
                        >
                            <option value="">No team</option>
                            {teams.map((t) => (
                                <option key={t.slug} value={t.slug}>
                                    {t.name}
                                </option>
                            ))}
                        </select>
                        {createError && (
                            <p className="mt-3 text-xs text-red-400/80">{createError}</p>
                        )}
                        <div className="mt-6 flex justify-end gap-3">
                            <button
                                className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-50"
                                disabled={creating}
                                onClick={() => setShowNew(false)}
                            >
                                Cancel
                            </button>
                            <button
                                className="text-foreground/90 flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.1] px-4 py-2 text-sm transition-colors hover:bg-white/[0.16] disabled:opacity-50"
                                disabled={creating || !newName.trim()}
                                onClick={handleCreate}
                            >
                                {creating ? (
                                    <>
                                        <div className="border-foreground/30 border-t-foreground/80 h-3 w-3 animate-spin rounded-full border-2" />
                                        Creating…
                                    </>
                                ) : (
                                    <>
                                        <IconCheck size={14} />
                                        Create
                                    </>
                                )}
                            </button>
                        </div>
                        <button
                            aria-label="Close"
                            className="text-muted-foreground/30 hover:text-foreground/60 absolute top-4 right-4 transition-colors disabled:opacity-30"
                            disabled={creating}
                            onClick={() => !creating && setShowNew(false)}
                        >
                            <IconX size={16} />
                        </button>
                    </div>
                </div>
            )}
            {/* Team create/edit dialog */}
            {showTeamDialog && (
                <div className="fixed inset-0 z-50 flex items-center justify-center">
                    <div
                        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
                        onClick={() => !savingTeam && setShowTeamDialog(false)}
                    />
                    <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-md rounded-2xl border border-white/[0.08] p-6 shadow-2xl backdrop-blur-md">
                        <h3 className="font-heading text-foreground/90 mb-1 text-lg">
                            {editingTeam ? 'Edit Team' : 'New Team'}
                        </h3>
                        <p className="text-muted-foreground/50 mb-5 text-sm">
                            {editingTeam
                                ? `Editing ${editingTeam.name}`
                                : 'Create a new team to group agents together.'}
                        </p>
                        <div className="space-y-3">
                            <div>
                                <label className="text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase">
                                    Name
                                </label>
                                <input
                                    autoFocus
                                    className="text-foreground/80 placeholder:text-muted-foreground/30 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none"
                                    onChange={(e) => setTeamName(e.target.value)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter') {
                                            handleSaveTeam();
                                        }
                                    }}
                                    placeholder="e.g. Matrix Ops"
                                    value={teamName}
                                />
                            </div>
                            <div>
                                <label className="text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase">
                                    Color
                                </label>
                                <input
                                    className="text-foreground/80 placeholder:text-muted-foreground/30 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 font-mono text-sm transition-colors focus:border-white/[0.16] focus:outline-none"
                                    onChange={(e) => setTeamColor(e.target.value)}
                                    placeholder="#8B5CF6 or purple"
                                    value={teamColor}
                                />
                            </div>
                            <div>
                                <label className="text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase">
                                    Description{' '}
                                    <span className="text-muted-foreground/25 tracking-normal normal-case">
                                        (optional)
                                    </span>
                                </label>
                                <input
                                    className="text-foreground/80 placeholder:text-muted-foreground/30 w-full rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.16] focus:outline-none"
                                    onChange={(e) => setTeamDesc(e.target.value)}
                                    placeholder="What this team does…"
                                    value={teamDesc}
                                />
                            </div>
                        </div>
                        <div className="mt-6 flex items-center justify-between">
                            <div>
                                {editingTeam && (
                                    <button
                                        className="text-[11px] text-red-400/60 transition-colors hover:text-red-400"
                                        onClick={() => {
                                            handleDeleteTeam(editingTeam.slug);
                                            setShowTeamDialog(false);
                                        }}
                                    >
                                        Delete team
                                    </button>
                                )}
                            </div>
                            <div className="flex gap-3">
                                <button
                                    className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-50"
                                    disabled={savingTeam}
                                    onClick={() => setShowTeamDialog(false)}
                                >
                                    Cancel
                                </button>
                                <button
                                    className="text-foreground/90 flex items-center gap-2 rounded-lg border border-white/[0.08] bg-white/[0.1] px-4 py-2 text-sm transition-colors hover:bg-white/[0.16] disabled:opacity-50"
                                    disabled={savingTeam || !teamName.trim()}
                                    onClick={handleSaveTeam}
                                >
                                    {(() => {
                                        if (savingTeam) {
                                            return 'Saving…';
                                        }
                                        return editingTeam ? 'Save' : 'Create';
                                    })()}
                                </button>
                            </div>
                        </div>
                        <button
                            aria-label="Close"
                            className="text-muted-foreground/30 hover:text-foreground/60 absolute top-4 right-4 transition-colors"
                            disabled={savingTeam}
                            onClick={() => !savingTeam && setShowTeamDialog(false)}
                        >
                            <IconX size={16} />
                        </button>
                    </div>
                </div>
            )}
        </Page>
    );
}
