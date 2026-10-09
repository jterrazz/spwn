import { describe, expect, test } from 'vitest';

import type { World } from '../../../domain/model';
import { byTeam, countsOf, filtered, rosterOf } from '../roster';

const MATRIX: World = {
    agent: '',
    agents: [{ name: 'neo', role: 'chief', status: 'running' }],
    config: 'matrix',
    created_at: '',
    id: 'matrix',
    status: 'running',
};

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
        // Given - the same three agents, ops a known team
        const roster = rosterOf(
            [MATRIX],
            [{ name: 'trinity', team: 'ops' }, { name: 'neo' }, { name: 'morpheus' }],
        );

        // Then - what each view keeps of it
        expect({
            counts: countsOf(roster),
            deployed: filtered(roster, 'deployed', '').map((agent) => agent.name),
            groups: byTeam(roster, [{ name: 'Operations', slug: 'ops' }]).map((group) => [
                group.team?.name ?? 'none',
                group.agents.map((agent) => agent.name),
            ]),
            searched: filtered(roster, 'all', 'TRI').map((agent) => agent.name),
        }).toStrictEqual({
            counts: { all: 3, deployed: 1, limbo: 2 },
            deployed: ['neo'],
            groups: [
                ['Operations', ['trinity']],
                ['none', ['morpheus', 'neo']],
            ],
            searched: ['trinity'],
        });
    });
});
