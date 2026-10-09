import { getWorldName } from '../../domain/model';
import type { Team, World } from '../../domain/model';

export interface ListedAgent {
    name: string;
    team?: string;
    role?: string;
}

export interface RosterAgent {
    name: string;
    role: string;
    team?: string | undefined;
    status: string;
    worldId?: string | undefined;
    worldName?: string | undefined;
}

export type StatusFilter = 'all' | 'deployed' | 'limbo';

export interface TeamGroup {
    team: null | Team;
    agents: RosterAgent[];
}

/** Every agent the project knows, placed in the world that deploys it — or in limbo — by name. */
export function rosterOf(worlds: World[], agents: ListedAgent[]): RosterAgent[] {
    const placements = new Map(
        worlds.flatMap((world) =>
            world.agents.map((agent) => [agent.name, { agent, world }] as const),
        ),
    );
    return agents
        .map((agent) => {
            const placed = placements.get(agent.name);
            return {
                name: agent.name,
                role: placed?.agent.role ?? agent.role ?? 'worker',
                status: placed?.agent.status ?? 'limbo',
                team: agent.team,
                worldId: placed?.world.id,
                worldName: placed === undefined ? undefined : getWorldName(placed.world),
            };
        })
        .toSorted((a, b) => a.name.localeCompare(b.name));
}

function deployed(agent: RosterAgent): boolean {
    return agent.worldId !== undefined;
}

export function filtered(
    agents: RosterAgent[],
    filter: StatusFilter,
    query: string,
): RosterAgent[] {
    const needle = query.trim().toLowerCase();
    return agents.filter(
        (agent) =>
            (filter === 'all' || deployed(agent) === (filter === 'deployed')) &&
            agent.name.toLowerCase().includes(needle),
    );
}

export function countsOf(agents: RosterAgent[]): Record<StatusFilter, number> {
    const placed = agents.filter(deployed).length;
    return { all: agents.length, deployed: placed, limbo: agents.length - placed };
}

/** The agents grouped by team, teams by name, the agents with no team last. */
export function byTeam(agents: RosterAgent[], teams: Team[]): TeamGroup[] {
    const members = new Map<string, RosterAgent[]>();
    const solo: RosterAgent[] = [];
    for (const agent of agents) {
        if (agent.team === undefined || agent.team === '') {
            solo.push(agent);
        } else {
            members.set(agent.team, [...(members.get(agent.team) ?? []), agent]);
        }
    }
    const groups = [...members].map(([slug, list]) => ({
        agents: list,
        team: teams.find((team) => team.slug === slug) ?? { name: slug, slug },
    }));
    const sorted: TeamGroup[] = groups.toSorted((a, b) => a.team.name.localeCompare(b.team.name));
    return solo.length > 0 ? [...sorted, { agents: solo, team: null }] : sorted;
}
