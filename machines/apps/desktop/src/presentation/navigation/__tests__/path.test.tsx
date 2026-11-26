import {
    banner,
    button,
    component,
    content,
    field,
    navigation,
    option,
    within,
} from '@jterrazz/test';
import type { ComponentVisitor } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { wordOfSent } from '../../words/engine-words';
import { NO_STATE_WORD } from '../../words/words';
import { machineRoute } from '../route';
import type { PlainPage } from '../route';
import { PLACES } from '../segments';
import { TITLES } from '../titles';
import { Window } from '../window';
import { PINNED } from './pinned';

const LISTS = import.meta.glob<unknown>('../../../fixtures/worlds/*/machines.list.json', {
    eager: true,
    import: 'default',
});

type Fleet = {
    world: string;
    macs: readonly { name: string; vms: readonly string[] }[];
    rows: readonly { name: string; row: string }[];
};

function fieldOf(value: unknown, key: string): unknown {
    if (typeof value !== 'object' || value === null) {
        return undefined;
    }
    const read: unknown = Object.getOwnPropertyDescriptor(value, key)?.value;
    return read;
}

function listOf(value: unknown): unknown[] {
    return Array.isArray(value) ? value : [];
}

function nameOf(value: unknown): string {
    const name = fieldOf(value, 'Name');
    return typeof name === 'string' ? name : '';
}

// value: a machine's row as the column names it: its name, then its standing's word.
function rowOf(
    machine: unknown,
    of: 'MachinesMacStanding' | 'MachinesVMStanding',
): { name: string; row: string } {
    const standing = fieldOf(machine, 'Standing');
    const word = typeof standing === 'string' ? wordOfSent(of, standing) : NO_STATE_WORD;
    return { name: nameOf(machine), row: `${nameOf(machine)} ${word}` };
}

function fleetOf(path: string, recorded: unknown): Fleet {
    const [first] = listOf(fieldOf(recorded, 'exchanges'));
    const out = listOf(fieldOf(first, 'answers'))
        .map((answer) => fieldOf(answer, 'out'))
        .findLast((found) => found !== undefined);
    const listed = listOf(fieldOf(fieldOf(out, 'All'), 'Macs'));
    const macs = listed.map((mac) => ({
        name: nameOf(mac),
        vms: listOf(fieldOf(mac, 'VMs')).map(nameOf),
    }));
    const rows = listed.flatMap((mac) => [
        rowOf(mac, 'MachinesMacStanding'),
        ...listOf(fieldOf(mac, 'VMs')).map((vm) => rowOf(vm, 'MachinesVMStanding')),
    ]);
    return { macs, rows, world: path.split('/').at(-2) ?? '' };
}

const FLEETS = Object.entries(LISTS)
    .map(([path, recorded]) => fleetOf(path, recorded))
    .filter((fleet) => fleet.macs.length > 0);
const recordingFor = await recordingsOf(...FLEETS.map((fleet) => fleet.world));

const path = navigation('Path');
const column = navigation('Column');

const bar = banner();

// value: the Library's pages, in the column's order.
const LIBRARY: readonly PlainPage[] = [
    'catalog-apps',
    'catalog-skills',
    'catalog-settings',
    'machine-files',
    'storage',
    'passwords',
];

// value: a level's popover opened by its segment, filtered down to one name, chosen by Enter; a machine's segment
// value: carries its state's words after its name, a place's its name alone.
async function choose(
    visitor: ComponentVisitor,
    {
        segment,
        filter,
        name,
        exact,
    }: { segment: string; filter: string; name: string; exact: boolean },
): Promise<void> {
    await visitor.click(within(path, button(segment, { exact })));
    await visitor.fill(field(filter), name);
    await visitor.press('Enter');
    await visitor.see(within(path, button(name, { exact })));
}

describe('PathNav', () => {
    test.each(FLEETS)(
        'every page of $world is reached through the path bar',
        async ({ world, macs, rows }) => {
            // Given - the window on Home over the world
            const result = await component.render(
                <EngineProvider dependencies={composeReplay({ recording: recordingFor(world) })}>
                    <Window initial={{ page: 'home' }} memory={PINNED} pages={PAGES} />
                </EngineProvider>,
                async (visitor) => {
                    await visitor.see(within(path, content(TITLES.home)));

                    // When - Storage is opened from the column, then every place from the place segment's popover
                    await visitor.click(within(column, button(TITLES.storage, { exact: false })));
                    await visitor.see(within(path, button(TITLES.storage)));
                    let at: string = TITLES.storage;
                    for (const place of PLACES) {
                        await choose(visitor, {
                            exact: true,
                            filter: 'Filter places',
                            name: TITLES[place],
                            segment: at,
                        });
                        at = TITLES[place];
                    }

                    // When - the first Mac is opened from the column, then every machine from its Mac's segment
                    const [first] = macs;
                    await visitor.click(within(column, button(rows[0]?.row ?? '')));
                    let on = first?.name ?? '';
                    for (const mac of macs) {
                        for (const machine of [mac.name, ...mac.vms]) {
                            await choose(visitor, {
                                exact: false,
                                filter: 'Filter machines',
                                name: machine,
                                segment: on,
                            });
                            on = mac.name;
                        }
                    }

                    // Then - Inbox, Settings and Home are a press away in the bar
                    await visitor.click(within(bar, button('Inbox', { exact: false })));
                    await visitor.see(within(path, content(TITLES.inbox)));
                    await visitor.click(within(bar, button('Settings')));
                    await visitor.see(within(path, content(TITLES.settings)));
                    await visitor.click(within(bar, button('Home')));
                    await visitor.see(within(path, content(TITLES.home)));
                },
            );

            // Then - the walk ended on Home, the window's first page
            expect(result.html.text).toContain(TITLES.home);
        },
        120_000,
    );

    test.each(FLEETS)(
        'every machine and every Library page of $world is reached through the column',
        async ({ world, rows }) => {
            // Given - the window on Home over the world, its column pinned
            const result = await component.render(
                <EngineProvider dependencies={composeReplay({ recording: recordingFor(world) })}>
                    <Window initial={{ page: 'home' }} memory={PINNED} pages={PAGES} />
                </EngineProvider>,
                async (visitor) => {
                    await visitor.see(within(path, content(TITLES.home)));

                    // When - each machine, then each Library page, is pressed in the column
                    for (const { name, row } of rows) {
                        await visitor.click(within(column, button(row)));
                        await visitor.see(within(path, button(name, { exact: false })));
                    }
                    for (const page of LIBRARY) {
                        await visitor.click(within(column, button(TITLES[page], { exact: false })));
                        await visitor.see(within(path, button(TITLES[page])));
                    }
                },
            );

            // Then - the walk ended on the Library's last page, the column still beside it
            expect(result.html.text).toContain(TITLES.passwords);
        },
        120_000,
    );

    test('a path too long for a 640 px window folds its middle, and the fold reaches what it hides', async () => {
        // Given - the window on linux-1, a VM of macbook-pro, over two-macs, 640 px wide
        const result = await component.viewport({ height: 800, width: 640 }).render(
            <EngineProvider dependencies={composeReplay({ recording: recordingFor('two-macs') })}>
                <Window initial={machineRoute('linux-1')} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                // When - the path is read
                await visitor.see(within(path, button('Folded levels')));

                // Then - Fleet and the VM keep their names, the Mac is behind the fold, and the fold opens it
                await visitor.see(within(path, button('Fleet')));
                await visitor.see(within(path, button('linux-1', { exact: false })));
                await visitor.gone(within(path, button('macbook-pro', { exact: false })));
                await visitor.click(within(path, button('Folded levels')));
                await visitor.click(option('macbook-pro', { exact: false }));
                await visitor.see(within(path, button('macbook-pro', { exact: false })));
            },
        );

        // Then - the Mac, once chosen from the fold, is drawn in the path
        expect(result.html.text).toContain('macbook-pro');
    });

    test('the same path in a 1440 px bar folds nothing', async () => {
        // Given - the window on linux-1 over two-macs, 1440 px wide
        const result = await component.viewport({ height: 900, width: 1440 }).render(
            <EngineProvider dependencies={composeReplay({ recording: recordingFor('two-macs') })}>
                <Window initial={machineRoute('linux-1')} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                // When - the path is read
                await visitor.see(within(path, button('linux-1', { exact: false })));

                // Then - every level keeps its name, and no fold is drawn
                await visitor.see(within(path, button('macbook-pro', { exact: false })));
                await visitor.gone(within(path, button('Folded levels')));
            },
        );

        // Then - the fold is nowhere in the page
        expect(result.html.text).not.toContain('Folded levels');
    });
});
