import type { CoreGuideGuide } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { guidePropsOf, routeOfNav } from '../guide-of';

const SOURCES = import.meta.glob<string>(['../*.ts', '../*.tsx', '../../../navigation/opened.ts'], {
    eager: true,
    import: 'default',
    query: '?raw',
});

const DECIDED = [
    /['"](?:done|now|waiting|failed|next|interrupted)['"]/u,
    /\.(?:status|ended|move|live|illustration)\s*(?:===|!==)\s*['"`]/u,
    /\bswitch\s*\(\s*[\w.]*(?:status|ended|move)/u,
    /\bsteps\s*\.\s*(?:filter|map|find|some|every|reduce)\b/u,
];

const INTERRUPTED: CoreGuideGuide = {
    act: 'New machine',
    ended: 'interrupted',
    id: 'machines.create:linux-4',
    line: 'Creating linux-4 was interrupted at Boot',
    live: false,
    steps: [
        { id: 'cloning', label: 'Disk', status: 'done' },
        { detail: 'Interrupted', id: 'booting', label: 'Boot', status: 'failed' },
        { id: 'address', label: 'Network', status: 'next' },
        { id: 'ssh', label: 'SSH', status: 'next' },
    ],
    subject: 'linux-4',
    view: {
        action: {
            args: { VM: 'linux-4' },
            call: 'machines.start',
            label: 'Start linux-4',
            on: 'macbook-pro',
        },
        headline: { next: '', now: 'Boot was interrupted' },
        illustration: 'boot-failed',
        kept: 'linux-4 is kept, stopped.',
        move: 'owner',
        sentence: 'The engine stopped while it created linux-4.',
    },
};

const NOTHING = { action: undefined, onHide: () => undefined, quiet: undefined };

describe('guide-of', () => {
    test('an interrupted act is drawn as the engine serves it, its ending in the engine’s word', () => {
        // Given - an interrupted Guide as the engine serves it
        const guide = INTERRUPTED;

        // When - it is mapped, its action already a press
        const props = guidePropsOf(guide, {
            ...NOTHING,
            action: { label: 'Start linux-4', onPress: () => undefined },
        });

        // Then - the steps and the view are the engine's, the ending its word
        expect(props).toMatchObject({
            act: 'New machine',
            ended: 'Interrupted',
            steps: guide.steps,
            view: {
                action: { label: 'Start linux-4' },
                headline: guide.view.headline,
                illustration: 'boot-failed',
                kept: 'linux-4 is kept, stopped.',
                move: 'owner',
                sentence: 'The engine stopped while it created linux-4.',
            },
        });
    });

    test('an act still running has no ending', () => {
        // Given - a live Guide
        const running: CoreGuideGuide = {
            act: INTERRUPTED.act,
            id: INTERRUPTED.id,
            line: 'Creating linux-4: Boot',
            live: true,
            steps: INTERRUPTED.steps,
            subject: INTERRUPTED.subject,
            view: INTERRUPTED.view,
        };

        // When - it is mapped
        const props = guidePropsOf(running, NOTHING);

        // Then - it carries none
        expect(props.ended).toBeUndefined();
    });

    test('a machine’s page opens that machine, any other page nothing', () => {
        // Given - the pages a Guide may name
        const pages = ['machine:linux-2', 'machine:', 'ports'];

        // When - each is routed
        const routes = pages.map((page) => routeOfNav(page));

        // Then - only the machine's opens
        expect(routes).toStrictEqual([
            { machine: 'linux-2', page: 'machine', tab: 'Overview' },
            undefined,
            undefined,
        ]);
    });

    test('the Guide decides nothing: its files name no status, no ending and no move, and compute no step', () => {
        // Given - every file that draws the Guide or opens it
        const files = Object.entries(SOURCES);

        // When - each is read for a comparison of a status, an ending or a move, or a walk of the steps
        const decided = files.flatMap(([name, source]) =>
            DECIDED.filter((pattern) => pattern.test(source)).map(
                (pattern) => `${name} matches ${String(pattern)}`,
            ),
        );

        // Then - none does
        expect(decided).toStrictEqual([]);
    });
});
