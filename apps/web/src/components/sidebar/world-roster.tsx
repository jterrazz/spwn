'use client';

import {
    IconBoltFilled,
    IconBookFilled,
    IconCircleFilled,
    IconHomeFilled,
    IconMessageFilled,
    IconMoonFilled,
} from '@tabler/icons-react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Suspense } from 'react';

import type { Agent, Team, World } from '../../domain/model';
import { useList } from '../../hooks/use-list';
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from '../ui/sidebar';

interface AgentIcon {
    icon: typeof IconBoltFilled;
    color: string;
    dim: boolean;
}

const STOPPED: AgentIcon = { color: 'text-zinc-500/30', dim: true, icon: IconCircleFilled };

const AGENT_ICONS: Record<string, AgentIcon> = {
    idle: { color: 'text-amber-400/50', dim: true, icon: IconCircleFilled },
    running: { color: 'text-green-400', dim: false, icon: IconBoltFilled },
    sleeping: { color: 'text-purple-400', dim: false, icon: IconMoonFilled },
    stopped: STOPPED,
    waiting: { color: 'text-amber-400 animate-pulse', dim: false, icon: IconMessageFilled },
};

interface TeamGroup {
    team: Team;
    agents: Agent[];
}

function grouped(agents: Agent[], teams: Team[]): { groups: TeamGroup[]; solo: Agent[] } {
    const teamOf = new Map(
        teams.flatMap((team) => (team.members ?? []).map((member) => [member, team] as const)),
    );
    const groups = new Map<string, TeamGroup>();
    const solo: Agent[] = [];
    for (const agent of agents) {
        const team = teamOf.get(agent.name);
        if (team === undefined) {
            solo.push(agent);
        } else {
            const group = groups.get(team.slug) ?? { agents: [], team };
            group.agents.push(agent);
            groups.set(team.slug, group);
        }
    }
    return { groups: [...groups.values()], solo };
}

function AgentEntry({ agent, world }: { agent: Agent; world: World }) {
    const router = useRouter();
    const pathname = usePathname();
    const inWorld = useSearchParams().get('world');
    const { color, dim, icon: StatusIcon } = AGENT_ICONS[agent.status] ?? STOPPED;
    return (
        <SidebarMenuItem>
            <SidebarMenuButton
                isActive={
                    decodeURIComponent(pathname) === `/agents/${agent.name}` && inWorld === world.id
                }
                onClick={() => {
                    router.push(`/agents/${encodeURIComponent(agent.name)}?world=${world.id}`);
                }}
            >
                <span className="-mx-[2px] flex h-[20px] w-[20px] shrink-0 -translate-x-[0.5px] items-center justify-center rounded-full bg-white/[0.15]">
                    <StatusIcon className={`!size-[12px] ${color}`} />
                </span>
                <span className={dim ? 'opacity-50' : ''}>{agent.name}</span>
            </SidebarMenuButton>
        </SidebarMenuItem>
    );
}

function Agents({ world }: { world: World }) {
    const teams = useList<Team>('/api/teams');
    if (world.agents.length === 0) {
        return <p className="text-muted-foreground/25 px-2 py-1.5 text-xs">No agents deployed</p>;
    }
    const { groups, solo } = grouped(world.agents, teams);
    const entries = (agents: Agent[]) =>
        agents.map((agent) => <AgentEntry agent={agent} key={agent.name} world={world} />);
    if (groups.length === 0) {
        return <>{entries(solo)}</>;
    }
    return (
        <>
            {groups.map(({ team, agents }) => (
                <div className="mt-1" key={team.slug}>
                    <p className="text-sidebar-foreground/25 flex items-center gap-1.5 px-2 py-1 text-[9px] tracking-[0.12em] uppercase">
                        <span style={(team.color ?? '') === '' ? undefined : { color: team.color }}>
                            {team.name}
                        </span>
                    </p>
                    {entries(agents)}
                </div>
            ))}
            {solo.length > 0 && (
                <div className="mt-1">
                    <p className="text-sidebar-foreground/20 px-2 py-1 text-[9px] tracking-[0.12em] uppercase">
                        No team
                    </p>
                    {entries(solo)}
                </div>
            )}
        </>
    );
}

/** The selected world's own destinations — its home, its knowledge — and its agents, grouped by team. */
export function WorldRoster({ world }: { world: World }) {
    const router = useRouter();
    const pathname = usePathname();
    const places = [
        { href: `/world/${world.id}`, icon: <IconHomeFilled size={16} />, label: 'Home' },
        {
            href: `/world/${world.id}/knowledge`,
            icon: <IconBookFilled size={16} />,
            label: 'Knowledge',
        },
    ];
    return (
        <SidebarMenu className="mt-3">
            {places.map((place) => (
                <SidebarMenuItem key={place.href}>
                    <SidebarMenuButton
                        isActive={pathname === place.href}
                        onClick={() => {
                            router.push(place.href);
                        }}
                    >
                        {place.icon}
                        <span>{place.label}</span>
                    </SidebarMenuButton>
                </SidebarMenuItem>
            ))}
            <Suspense fallback={null}>
                <Agents world={world} />
            </Suspense>
        </SidebarMenu>
    );
}
