import { button, component, content } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { FleetBoard } from '../fleet-board';
import type { FleetMac } from '../fleet-board';

const RUNNING = { tone: 'ok', word: 'running' } as const;

const MACS: readonly FleetMac[] = [
    {
        cpu: '22 %',
        free: '28 free',
        kind: '2 machines',
        machines: [
            {
                cpu: '1 %',
                identity: 'blue',
                memory: '5.3 / 8',
                name: 'agent-7f',
                purpose: 'Write the changelog',
                sentence: 'Running. Nothing to do.',
                share: { at: 0.0833, used: 0.66, width: 0.1667 },
                state: RUNNING,
            },
            {
                cpu: '9 %',
                identity: 'teal',
                line: 'Apply it from its Packages tab.',
                memory: '5.3 / 8',
                name: 'linux-1',
                purpose: 'Settings from a machine file',
                sentence: 'Its machine file changed: 2 apps are waiting to be applied.',
                share: { at: 0.25, used: 0.66, width: 0.1667 },
                state: { tone: 'warn', word: 'needs you' },
            },
        ],
        name: 'macbook-pro',
        state: { tone: 'warn', word: '1 needs you' },
        system: { at: 0, used: 1, width: 0.0833 },
        thisMac: true,
    },
    {
        cpu: '36 %',
        free: '36 free',
        kind: '1 machine',
        machines: [
            {
                cpu: '10 %',
                identity: 'pink',
                memory: '16 / 24',
                name: 'infra',
                purpose: "The cluster's control plane",
                sentence: 'Running. Nothing to do.',
                share: { at: 0.0625, used: 0.67, width: 0.375 },
                state: RUNNING,
            },
        ],
        name: 'mac-mini',
        state: { tone: 'ok', word: 'all running' },
        system: { at: 0, used: 1, width: 0.0625 },
        thisMac: false,
    },
];

const MINI_CARD = 'mac-mini · 1 machine · all running';
const PRO_CARD = 'macbook-pro · 2 machines · 1 needs you';

type Seen = { rows: string[]; pressed: (null | string)[]; expanded: (null | string)[] };

function seen(): Seen {
    const machines = [...document.querySelectorAll('[aria-expanded]')];
    return {
        expanded: machines.map((one) => one.getAttribute('aria-expanded')),
        pressed: [...document.querySelectorAll('[aria-pressed]')].map((one) =>
            one.getAttribute('aria-pressed'),
        ),
        rows: machines.map((one) => one.textContent),
    };
}

function board(onOpen: (name: string) => void = () => undefined) {
    return (
        <FleetBoard
            macs={MACS}
            onOpen={onOpen}
            onOpenMac={() => undefined}
            summary="3 machines · 2 Macs"
            thisMac="this Mac"
        />
    );
}

describe('FleetBoard', () => {
    test('a Mac pressed keeps its rows alone, and pressed again shows every Mac', async () => {
        // Given - a board of two Macs, three machines in all
        const states: Seen[] = [];
        await component.render(board(), async (visitor) => {
            // When - the mini's card is pressed, then pressed again
            states.push(seen());
            await visitor.click(button(MINI_CARD));
            states.push(seen());
            await visitor.click(button(MINI_CARD));
            states.push(seen());
        });

        // Then - its one row alone while it is pressed, every row before and after
        expect(states).toStrictEqual([
            {
                expanded: ['false', 'false', 'false'],
                pressed: ['false', 'false'],
                rows: ['agent-7f', 'linux-1', 'infra'],
            },
            { expanded: ['false'], pressed: ['false', 'true'], rows: ['infra'] },
            {
                expanded: ['false', 'false', 'false'],
                pressed: ['false', 'false'],
                rows: ['agent-7f', 'linux-1', 'infra'],
            },
        ]);
    });

    test('a row pressed opens on its sentence and its Open, which opens that machine', async () => {
        // Given - a board whose Open is recorded
        const opened: string[] = [];
        const states: Seen[] = [];
        const result = await component.render(
            board((name) => {
                opened.push(name);
            }),
            async (visitor) => {
                // When - linux-1's row is pressed, then its Open
                await visitor.click(button('linux-1'));
                await visitor.see(
                    content('Its machine file changed: 2 apps are waiting to be applied.'),
                );
                states.push(seen());
                await visitor.click(button('Open linux-1'));
            },
        );

        // Then - linux-1 alone is expanded, its sentence and line are shown, and Open named it
        expect(states).toStrictEqual([
            {
                expanded: ['false', 'true', 'false'],
                pressed: ['false', 'false'],
                rows: ['agent-7f', 'linux-1', 'infra'],
            },
        ]);
        expect(result.html.text).toContain('Apply it from its Packages tab.');
        expect(opened).toStrictEqual(['linux-1']);
    });

    test('a row pressed again closes, and a Mac chosen closes the row left open', async () => {
        // Given - a board with linux-1 opened
        const states: Seen[] = [];
        await component.render(board(), async (visitor) => {
            // When - linux-1 is pressed twice, then opened and the macbook-pro chosen
            await visitor.click(button('linux-1'));
            await visitor.click(button('linux-1'));
            states.push(seen());
            await visitor.click(button('linux-1'));
            await visitor.click(button(PRO_CARD));
            states.push(seen());
        });

        // Then - closed by its second press, and closed when its Mac was chosen
        expect(states).toStrictEqual([
            {
                expanded: ['false', 'false', 'false'],
                pressed: ['false', 'false'],
                rows: ['agent-7f', 'linux-1', 'infra'],
            },
            {
                expanded: ['false', 'false'],
                pressed: ['true', 'false'],
                rows: ['agent-7f', 'linux-1'],
            },
        ]);
    });
});
