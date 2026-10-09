import { describe, expect, test } from 'vitest';

import type { World } from '../../../domain/model';
import { byTeam, countsOf, filtered, rosterOf } from '../roster';
import type { RosterAgent } from '../roster';

const MATRIX: World = {
    agent: '',
    agents: [{ name: 'neo', role: 'chief', status: 'running' }],
    config: 'matrix',
    created_at: '',
    id: 'matrix',
    status: 'running',
};

const TEAMS = [
    { name: 'Design', slug: 'design' },
    { name: 'Operations', slug: 'ops' },
];

function names(agents: RosterAgent[]): string[] {
    return agents.map((agent) => agent.name);
}

describe('roster', () => {
    test('each agent is placed in the world that deploys it, the rest in limbo', () => {
        // Given - neo deployed in matrix, trinity and morpheus in no world
        const roster = rosterOf(
            [MATRIX],
            [
                { name: 'trinity', team: 'ops' },
                { name: 'neo' },
                { name: 'morpheus', role: 'manager' },
            ],
        );

        // Then - by name, each with its world, role and state
        expect(roster).toStrictEqual([
            {
                name: 'morpheus',
                role: 'manager',
                status: 'limbo',
                team: undefined,
                worldId: undefined,
                worldName: undefined,
            },
            {
                name: 'neo',
                role: 'chief',
                status: 'running',
                team: undefined,
                worldId: 'matrix',
                worldName: 'matrix',
            },
            {
                name: 'trinity',
                role: 'worker',
                status: 'limbo',
                team: 'ops',
                worldId: undefined,
                worldName: undefined,
            },
        ]);
    });

    test('the filters, the counts and the team groups read the same roster', () => {
        // Given - five agents: trinity in ops, apoc in zion (a team with no page), oracle's team left empty
        const roster = rosterOf(
            [MATRIX],
            [
                { name: 'trinity', team: 'ops' },
                { name: 'neo' },
                { name: 'morpheus' },
                { name: 'apoc', team: 'zion' },
                { name: 'oracle', team: '' },
            ],
        );

        // Then - what each view keeps of it, teams by name and the agents of no team last
        expect({
            all: names(filtered(roster, 'all', '')),
            counts: countsOf(roster),
            deployed: names(filtered(roster, 'deployed', '')),
            groups: byTeam(roster, TEAMS).map((group) => [
                group.team?.name ?? 'none',
                names(group.agents),
            ]),
            limbo: names(filtered(roster, 'limbo', '')),
            searched: names(filtered(roster, 'all', ' TRI ')),
            teamless: byTeam([], TEAMS),
        }).toStrictEqual({
            all: ['apoc', 'morpheus', 'neo', 'oracle', 'trinity'],
            counts: { all: 5, deployed: 1, limbo: 4 },
            deployed: ['neo'],
            groups: [
                ['Operations', ['trinity']],
                ['zion', ['apoc']],
                ['none', ['morpheus', 'neo', 'oracle']],
            ],
            limbo: ['apoc', 'morpheus', 'oracle', 'trinity'],
            searched: ['trinity'],
            teamless: [],
        });
    });
});
