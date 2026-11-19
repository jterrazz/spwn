import { describe, expect, test } from 'vitest';

import { changeOf, endOf, routeOfLine } from '../model';

describe('model', () => {
    test('opens its machine on its Overview, and a package on its shelf', () => {
        // Given - a line about a machine, one about an app and one about a skill
        const machine = { Subject: { Kind: 'machine' as const, Name: 'infra' } };
        const app = { Subject: { Kind: 'app' as const, Name: 'node' } };
        const skill = { Subject: { Kind: 'skill' as const, Name: 'frontend-design' } };

        // When - each is routed
        const routes = [machine, app, skill, { Subject: null }].map((line) => routeOfLine(line));

        // Then - the machine's page, the two shelves, and nothing for a line with no subject
        expect(routes).toStrictEqual([
            { machine: 'infra', page: 'machine', tab: 'Overview' },
            { item: 'node', page: 'catalog-apps' },
            { item: 'frontend-design', page: 'catalog-skills' },
            undefined,
        ]);
    });

    test('says its versions when it moved a package, else its detail', () => {
        // Given - an upgrade and a held line
        const upgrade = { Detail: '', Upgrade: { From: '24.11.0', To: '24.11.1' } };
        const held = { Detail: 'updates itself', Upgrade: null };

        // When - each is worded
        const changes = [changeOf(upgrade), changeOf(held)];

        // Then - from and to, then the engine's sentence
        expect(changes).toStrictEqual(['24.11.0 → 24.11.1', 'updates itself']);
    });

    test('ends on who did it and why, else on the machines it touched', () => {
        // Given - a creation by a session, an upgrade over six machines and one over a single machine
        const created = { By: 'agent-7f', On: ['linux-3'], Purpose: 'review the docs lot' };
        const many = { By: '', On: ['a', 'b', 'c', 'd', 'e', 'f'], Purpose: '' };
        const one = { By: '', On: ['macbook-pro'], Purpose: '' };

        // When - each end is worded
        const ends = [endOf(created), endOf(many), endOf(one), endOf({ ...one, On: null })];

        // Then - who and the purpose, the count in the engine's noun, the one machine's name, and nothing
        expect(ends).toStrictEqual([
            'by agent-7f · review the docs lot',
            '6 machines',
            'macbook-pro',
            '',
        ]);
    });
});
