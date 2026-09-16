'use client';

import {
    IconAlertTriangleFilled,
    IconAssemblyFilled,
    IconBinaryTreeFilled,
    IconBoltFilled,
    IconBookFilled,
    IconBrandGithubFilled,
    IconCircleFilled,
    IconHexagonFilled,
    IconHomeFilled,
    IconBookFilled as IconKnowledgeFilled,
    IconMessageFilled,
    IconMoonFilled,
    IconSearch,
    IconSettingsFilled,
    IconUserFilled,
} from '@tabler/icons-react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

import {
    apiGet,
    getConnectionStatus,
    isGoApiAvailable,
    onConnectionStatusChange,
} from '@/api/client';
import type { ConnectionStatus } from '@/api/client';
import { DockerStatusPill } from '@/components/docker-status-pill';
import { ThemeToggle } from '@/components/theme-toggle';
import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarGroup,
    SidebarGroupLabel,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
} from '@/components/ui/sidebar';
import { UpgradeBanner } from '@/components/upgrade-banner';
import { WorldPlanet } from '@/components/world-planet';
import { getWorldName } from '@/domain/model';
import type { Team, World } from '@/domain/model';
import { useVersion } from '@/hooks/use-version';

interface StatusData {
    worlds: number;
    agents: number;
    running: number;
}
interface AppSidebarProps {
    worlds: World[];
    currentWorldId?: string | undefined;
    loading?: boolean | undefined;
    statusData?: null | StatusData | undefined;
}

type AgentIcon = { icon: typeof IconBoltFilled; color: string; dim: boolean };

/** What an unknown status renders as — also the `stopped` entry below. */
const AGENT_ICON_STOPPED: AgentIcon = {
    icon: IconCircleFilled,
    color: 'text-zinc-500/30',
    dim: true,
};

const AGENT_ICON: Record<string, AgentIcon> = {
    running: { icon: IconBoltFilled, color: 'text-green-400', dim: false },
    waiting: { icon: IconMessageFilled, color: 'text-amber-400 animate-pulse', dim: false },
    sleeping: { icon: IconMoonFilled, color: 'text-purple-400', dim: false },
    idle: { icon: IconCircleFilled, color: 'text-amber-400/50', dim: true },
    stopped: AGENT_ICON_STOPPED,
};

export function AppSidebar({
    worlds,
    currentWorldId,
    loading,
    statusData: _statusData,
}: AppSidebarProps) {
    const pathname = usePathname();
    const router = useRouter();
    const [worldsExpanded, setWorldsExpanded] = useState(false);
    const [teams, setTeams] = useState<Team[]>([]);

    const { version } = useVersion();

    // Fetch teams to group agents in the selected world by team
    useEffect(() => {
        apiGet<Team[]>('/api/teams')
            .then((t) => setTeams(t ?? []))
            .catch(() => {});
    }, []);
    const [connectionStatus, setConnectionStatus] =
        useState<ConnectionStatus>(getConnectionStatus());

    useEffect(() => {
        const unsub = onConnectionStatusChange(setConnectionStatus);
        const check = async () => {
            const goUp = await isGoApiAvailable();
            setConnectionStatus(goUp ? 'connected' : 'disconnected');
        };
        check();
        const interval = setInterval(check, 10_000);
        return () => {
            clearInterval(interval);
            unsub();
        };
    }, []);

    // Always show a world context (first world as default), but only highlight when on a world page
    const activeWorldId =
        currentWorldId || worlds.find((w) => pathname.startsWith(`/world/${w.id}`))?.id;
    const selectedWorldId = activeWorldId || worlds[0]?.id;
    const selectedWorld = selectedWorldId
        ? worlds.find((w) => w.id === selectedWorldId)
        : undefined;

    return (
        <Sidebar>
            {/* ── Header ── */}
            <SidebarHeader className="gap-0 pt-4 pb-4">
                <div className="flex items-center justify-between gap-2 px-2">
                    <div className="group/logo flex min-w-0 flex-1 items-center">
                        <Link className="flex items-center gap-1.5" href="/">
                            <span
                                className={`font-heading text-base transition-colors ${connectionStatus === 'connected' ? 'text-green-500' : 'text-red-400'}`}
                            >
                                ⬡
                            </span>
                            <span className="font-heading text-foreground text-base tracking-[0.12em]">
                                spwn
                            </span>
                        </Link>
                        <span
                            className={`ml-auto font-mono text-[10px] tracking-wider uppercase opacity-0 transition-opacity group-hover/logo:opacity-100 ${connectionStatus === 'connected' ? 'text-green-500/60' : 'text-red-400/60'}`}
                        >
                            {connectionStatus}
                        </span>
                    </div>
                    <button
                        aria-label="Search (⌘K)"
                        className="text-muted-foreground/30 hover:text-foreground flex h-8 w-8 shrink-0 items-center justify-center rounded-full transition-colors"
                        onClick={() =>
                            globalThis.dispatchEvent(
                                new KeyboardEvent('keydown', {
                                    key: 'k',
                                    metaKey: true,
                                    bubbles: true,
                                }),
                            )
                        }
                    >
                        <IconSearch size={16} stroke={2.5} />
                    </button>
                </div>
            </SidebarHeader>

            <SidebarContent>
                {/* ── Global ── */}
                <SidebarGroup className="pt-0">
                    <SidebarMenu>
                        <SidebarMenuItem>
                            <SidebarMenuButton
                                isActive={pathname === '/architect'}
                                onClick={() => router.push('/architect')}
                            >
                                <IconHexagonFilled size={16} />
                                <span>Architect</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                        <SidebarMenuItem>
                            <SidebarMenuButton
                                isActive={pathname === '/providers'}
                                onClick={() => router.push('/providers')}
                            >
                                <IconSettingsFilled size={16} />
                                <span>Settings</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    </SidebarMenu>
                </SidebarGroup>

                {/* ── Universe ── */}
                <SidebarGroup>
                    <SidebarGroupLabel className="text-sidebar-foreground/30 text-[10px] tracking-widest uppercase">
                        Universe
                    </SidebarGroupLabel>
                    <SidebarMenu>
                        <SidebarMenuItem>
                            <SidebarMenuButton
                                isActive={pathname === '/'}
                                onClick={() => router.push('/')}
                            >
                                <IconAssemblyFilled size={16} />
                                <span>Worlds</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                        <SidebarMenuItem>
                            <SidebarMenuButton
                                isActive={pathname === '/agents' || pathname.startsWith('/agents/')}
                                onClick={() => router.push('/agents')}
                            >
                                <IconUserFilled size={16} />
                                <span>Agents</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                        <SidebarMenuItem>
                            <SidebarMenuButton
                                isActive={pathname === '/tools'}
                                onClick={() => router.push('/tools')}
                            >
                                <IconBoltFilled size={16} />
                                <span>Tools</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                        <SidebarMenuItem>
                            <SidebarMenuButton
                                isActive={pathname === '/organizations'}
                                onClick={() => router.push('/organizations')}
                            >
                                <IconBinaryTreeFilled size={16} />
                                <span>Organizations</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    </SidebarMenu>
                </SidebarGroup>

                {/* ── Quick start hint ── */}
                {!loading && worlds.length === 0 && (
                    <SidebarGroup>
                        <div className="mx-2 space-y-2 rounded-lg border border-white/[0.06] bg-white/[0.02] px-3 py-3">
                            <p className="text-muted-foreground/30 text-[10px] tracking-widest uppercase">
                                Getting started
                            </p>
                            <div className="text-muted-foreground/40 space-y-1.5 text-[11px] leading-relaxed">
                                <p>
                                    <span className="text-foreground/50 font-mono">1.</span> Go to{' '}
                                    <button
                                        className="text-foreground/60 hover:text-foreground/80 underline decoration-white/10 underline-offset-2"
                                        onClick={() => router.push('/providers')}
                                    >
                                        Settings
                                    </button>{' '}
                                    and connect a provider
                                </p>
                                <p>
                                    <span className="text-foreground/50 font-mono">2.</span> Create
                                    an{' '}
                                    <button
                                        className="text-foreground/60 hover:text-foreground/80 underline decoration-white/10 underline-offset-2"
                                        onClick={() => router.push('/agents')}
                                    >
                                        Agent
                                    </button>
                                </p>
                                <p>
                                    <span className="text-foreground/50 font-mono">3.</span> Spawn a{' '}
                                    <button
                                        className="text-foreground/60 hover:text-foreground/80 underline decoration-white/10 underline-offset-2"
                                        onClick={() => router.push('/')}
                                    >
                                        World
                                    </button>
                                </p>
                            </div>
                        </div>
                    </SidebarGroup>
                )}

                {/* ── Worlds ── */}
                <SidebarGroup>
                    {worlds.length > 0 && selectedWorld ? (
                        <div className="space-y-1.5 px-1">
                            {/* Hero: current world */}
                            <button
                                className="bg-sidebar-accent/50 hover:bg-sidebar-accent flex w-full items-center gap-3 rounded-md px-2 py-2 text-left transition-colors"
                                onClick={() => router.push(`/world/${selectedWorld.id}`)}
                            >
                                <WorldPlanet size="md" world={selectedWorld} />
                                <span className="min-w-0 flex-1">
                                    <span className="text-foreground block truncate text-sm font-medium">
                                        {getWorldName(selectedWorld)}
                                    </span>
                                    <span className="text-muted-foreground/40 block text-[10px] tracking-widest uppercase">
                                        {selectedWorld.status} · {selectedWorld.agents.length} agent
                                        {selectedWorld.agents.length === 1 ? '' : 's'}
                                    </span>
                                </span>
                            </button>

                            {/* Quick-switch: other worlds. Condensed to one line with "+N" overflow; click to expand into a masonry wrap. */}
                            {(() => {
                                const others = worlds.filter((w) => w.id !== selectedWorld.id);
                                if (others.length === 0) {
                                    return null;
                                }

                                // Character-budget heuristic: fit as many pills on one line as reasonably possible.
                                // Each pill costs name.length + 3 (planet + padding/gap). Adjust budget if sidebar width changes.
                                const INLINE_CHAR_BUDGET = 22;
                                const inlineItems: typeof others = [];
                                const overflowItems: typeof others = [];
                                let budget = INLINE_CHAR_BUDGET;
                                for (const w of others) {
                                    const cost = getWorldName(w).length + 3;
                                    if (
                                        !worldsExpanded &&
                                        budget - cost < 0 &&
                                        inlineItems.length > 0
                                    ) {
                                        overflowItems.push(w);
                                    } else {
                                        inlineItems.push(w);
                                        budget -= cost;
                                    }
                                }
                                if (worldsExpanded) {
                                    overflowItems.length = 0; // All visible when expanded
                                }

                                const renderPill = (world: (typeof others)[number]) => (
                                    <button
                                        className="group/switch text-muted-foreground/50 hover:text-foreground hover:bg-sidebar-accent/40 flex h-6 shrink-0 items-center gap-1.5 rounded-md pr-2 pl-1 text-xs transition-colors"
                                        key={world.id}
                                        onClick={() => router.push(`/world/${world.id}`)}
                                    >
                                        <WorldPlanet
                                            className="opacity-80 transition-opacity group-hover/switch:opacity-100"
                                            size="sm"
                                            world={world}
                                        />
                                        <span>{getWorldName(world)}</span>
                                    </button>
                                );

                                const toggleBtn = (label: string) => (
                                    <button
                                        aria-label={
                                            worldsExpanded ? 'Collapse worlds' : 'Show all worlds'
                                        }
                                        className="text-muted-foreground/40 hover:text-foreground hover:bg-sidebar-accent/40 flex h-6 min-w-6 shrink-0 items-center justify-center rounded-md px-1.5 text-[11px] font-medium transition-colors"
                                        onClick={() => setWorldsExpanded((v) => !v)}
                                    >
                                        {label}
                                    </button>
                                );

                                return (
                                    <div
                                        className={`${
                                            worldsExpanded
                                                ? 'flex flex-wrap'
                                                : 'flex items-center overflow-hidden'
                                        } -mx-1 gap-1 px-1`}
                                    >
                                        {inlineItems.map(renderPill)}
                                        {!worldsExpanded &&
                                            overflowItems.length > 0 &&
                                            toggleBtn(`+${overflowItems.length}`)}
                                        {worldsExpanded && others.length > 0 && toggleBtn('Hide')}
                                    </div>
                                );
                            })()}
                        </div>
                    ) : (
                        <p className="text-muted-foreground/25 px-2 py-1.5 text-xs">
                            No worlds running
                        </p>
                    )}
                    {selectedWorld && (
                        <SidebarMenu className="mt-3">
                            <SidebarMenuItem>
                                <SidebarMenuButton
                                    isActive={pathname === `/world/${selectedWorld.id}`}
                                    onClick={() => router.push(`/world/${selectedWorld.id}`)}
                                >
                                    <IconHomeFilled size={16} />
                                    <span>Home</span>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                            <SidebarMenuItem>
                                <SidebarMenuButton
                                    isActive={pathname === `/world/${selectedWorld.id}/knowledge`}
                                    onClick={() =>
                                        router.push(`/world/${selectedWorld.id}/knowledge`)
                                    }
                                >
                                    <IconKnowledgeFilled size={16} />
                                    <span>Knowledge</span>
                                </SidebarMenuButton>
                            </SidebarMenuItem>

                            {(() => {
                                // Build agent-name → team lookup from teams data
                                const agentTeamMap = new Map<string, Team>();
                                for (const t of teams) {
                                    for (const m of t.members ?? []) {
                                        agentTeamMap.set(m, t);
                                    }
                                }

                                // Group world agents by team
                                const grouped = new Map<
                                    string,
                                    { team: null | Team; agents: typeof selectedWorld.agents }
                                >();
                                const soloAgents: typeof selectedWorld.agents = [];

                                for (const agent of selectedWorld.agents) {
                                    const team = agentTeamMap.get(agent.name);
                                    if (team) {
                                        const group = grouped.get(team.slug) ?? {
                                            team,
                                            agents: [],
                                        };
                                        group.agents.push(agent);
                                        grouped.set(team.slug, group);
                                    } else {
                                        soloAgents.push(agent);
                                    }
                                }

                                const renderAgent = (
                                    agent: (typeof selectedWorld.agents)[number],
                                ) => {
                                    const s = AGENT_ICON[agent.status] ?? AGENT_ICON_STOPPED;
                                    const StatusIcon = s.icon;
                                    return (
                                        <SidebarMenuItem key={agent.name}>
                                            <SidebarMenuButton
                                                isActive={
                                                    decodeURIComponent(pathname) ===
                                                        `/agents/${agent.name}` &&
                                                    typeof globalThis !== 'undefined' &&
                                                    new URLSearchParams(
                                                        globalThis.location.search,
                                                    ).get('world') === selectedWorld.id
                                                }
                                                onClick={() =>
                                                    router.push(
                                                        `/agents/${encodeURIComponent(agent.name)}?world=${selectedWorld.id}`,
                                                    )
                                                }
                                            >
                                                <span className="-mx-[2px] flex h-[20px] w-[20px] shrink-0 -translate-x-[0.5px] items-center justify-center rounded-full bg-white/[0.15]">
                                                    <StatusIcon
                                                        className={`!size-[12px] ${s.color}`}
                                                    />
                                                </span>
                                                <span className={s.dim ? 'opacity-50' : ''}>
                                                    {agent.name}
                                                </span>
                                            </SidebarMenuButton>
                                        </SidebarMenuItem>
                                    );
                                };

                                if (selectedWorld.agents.length === 0) {
                                    return (
                                        <p className="text-muted-foreground/25 px-2 py-1.5 text-xs">
                                            No agents deployed
                                        </p>
                                    );
                                }

                                // If no teams, render flat list (no headers)
                                if (grouped.size === 0) {
                                    return soloAgents.map(renderAgent);
                                }

                                // Render grouped: team sections + solo at bottom
                                return (
                                    <>
                                        {[...grouped.values()].map(({ team, agents }) => (
                                            <div className="mt-1" key={team!.slug}>
                                                <p className="text-sidebar-foreground/25 flex items-center gap-1.5 px-2 py-1 text-[9px] tracking-[0.12em] uppercase">
                                                    <span
                                                        style={
                                                            team!.color
                                                                ? { color: team!.color }
                                                                : undefined
                                                        }
                                                    >
                                                        {team!.name}
                                                    </span>
                                                </p>
                                                {agents.map(renderAgent)}
                                            </div>
                                        ))}
                                        {soloAgents.length > 0 && (
                                            <div className="mt-1">
                                                {grouped.size > 0 && (
                                                    <p className="text-sidebar-foreground/20 px-2 py-1 text-[9px] tracking-[0.12em] uppercase">
                                                        No team
                                                    </p>
                                                )}
                                                {soloAgents.map(renderAgent)}
                                            </div>
                                        )}
                                    </>
                                );
                            })()}
                        </SidebarMenu>
                    )}
                </SidebarGroup>
            </SidebarContent>

            {/* ── Footer ── */}
            <SidebarFooter>
                {version?.updateAvailable &&
                    version.latest !== process.env.NEXT_PUBLIC_APP_VERSION && (
                        <UpgradeBanner version={version} />
                    )}
                <div className="px-1.5 pb-1">
                    <DockerStatusPill />
                </div>
                <div className="flex items-center gap-1 px-1.5 pb-1">
                    <a
                        aria-label="Docs"
                        className="text-muted-foreground/30 hover:text-foreground flex h-8 w-8 items-center justify-center rounded-full transition-colors"
                        href="https://spwn.sh/docs"
                        target="_blank"
                    >
                        <IconBookFilled size={15} />
                    </a>
                    <a
                        aria-label="GitHub"
                        className="text-muted-foreground/30 hover:text-foreground flex h-8 w-8 items-center justify-center rounded-full transition-colors"
                        href="https://github.com/jterrazz/spwn"
                        target="_blank"
                    >
                        <IconBrandGithubFilled size={15} />
                    </a>
                    <a
                        aria-label="Feedback"
                        className="text-muted-foreground/30 hover:text-foreground flex h-8 w-8 items-center justify-center rounded-full transition-colors"
                        href="https://github.com/jterrazz/spwn/issues/new"
                        target="_blank"
                    >
                        <IconAlertTriangleFilled size={15} />
                    </a>
                    <div className="flex-1" />
                    <ThemeToggle />
                </div>
            </SidebarFooter>
        </Sidebar>
    );
}
