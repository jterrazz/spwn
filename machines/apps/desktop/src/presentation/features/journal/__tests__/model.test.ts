import { describe, expect, test } from 'vitest';

import { changeOf, endOf, keyOfLine, routeOfLine } from '../model';

describe('model', () => {
    test('opens its machine on its Overview, a package on its shelf, and an image on Storage', () => {
        // Given - a line about a machine, one about an app, one about a skill and one about an image
        const machine = { Subject: { Gone: false, Kind: 'machine' as const, Name: 'infra' } };
        const app = { Subject: { Gone: false, Kind: 'app' as const, Name: 'node' } };
        const skill = { Subject: { Gone: false, Kind: 'skill' as const, Name: 'frontend-design' } };
        const image = { Subject: { Gone: false, Kind: 'image' as const, Name: 'base-ubuntu' } };

        // When - each is routed
        const routes = [machine, app, skill, image, { Subject: null }].map((line) =>
            routeOfLine(line),
        );

        // Then - the machine's page, the two shelves, Storage where the images are listed, and nothing for no subject
        expect(routes).toStrictEqual([
            { machine: 'infra', page: 'machine', tab: 'Overview' },
            { item: 'node', page: 'catalog-apps' },
            { item: 'frontend-design', page: 'catalog-skills' },
            { page: 'storage' },
            undefined,
        ]);
    });

    test('opens nothing for a subject the engine says is gone', () => {
        // Given - a line about a machine deleted since, and one about a package the catalog no longer knows
        const machine = { Subject: { Gone: true, Kind: 'machine' as const, Name: 'gate-2' } };
        const app = { Subject: { Gone: true, Kind: 'app' as const, Name: 'old-tool' } };

        // When - each is routed
        const routes = [machine, app].map((line) => routeOfLine(line));

        // Then - neither opens a page
        expect(routes).toStrictEqual([undefined, undefined]);
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

    test('keys apart two failures of one night that the engine keeps per machine', () => {
        // Given - ripgrep failed at the same minute on two machines, each its own line
        const at = '2026-11-07T03:15:00Z';
        const lines = [
            { At: at, On: ['linux-1'], Text: 'ripgrep' },
            { At: at, On: ['api'], Text: 'ripgrep' },
        ];

        // When - each is keyed
        const keys = new Set(lines.map((line) => keyOfLine(line)));

        // Then - two keys
        expect(keys.size).toBe(2);
    });
});
