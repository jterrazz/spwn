import { button, component, dialog, field, focused, navigation, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { machineRoute } from '../route';
import type { Route } from '../route';
import { Window } from '../window';
import { PINNED } from './pinned';

const recordingFor = await recordingsOf('two-macs');

const path = navigation('Path');
const FILTER = field('Go to a section or a machine');
const LINUX_1 = within(path, button('linux-1', { exact: false }));
const FLEET = within(path, button('Fleet'));

// hazard: press wraps a key longer than one letter in braces, so a chord is user-event's hold and release
// hazard: with its outer braces left to press.
const COMMAND_L = 'Meta>}l{/Meta';

function windowOn(initial: Route) {
    return (
        <EngineProvider dependencies={composeReplay({ recording: recordingFor('two-macs') })}>
            <Window initial={initial} memory={PINNED} pages={PAGES} />
        </EngineProvider>
    );
}

// value: what each column of the open card says: its rows' names, or its empty sentence; one line when both are
// value: empty.
function columns(): string[] {
    const list = document.querySelector('[role="dialog"] [role="listbox"]');
    const groups = [...(list?.querySelectorAll('[role="group"]') ?? [])];
    if (groups.length === 0) {
        return [list?.textContent ?? ''];
    }
    return groups.map((group) => {
        const rows = [...group.querySelectorAll('[role="option"]')].map(
            (row) => row.querySelector('[class*="label"]')?.textContent ?? '',
        );
        const said = `${String(group.getAttribute('aria-label'))}: `;
        return (
            said + (rows.length > 0 ? rows.join(', ') : (group.lastElementChild?.textContent ?? ''))
        );
    });
}

function activeRow(): string {
    const input = document.querySelector('[role="dialog"] [role="combobox"]');
    const id = input?.getAttribute('aria-activedescendant') ?? '';
    return document.querySelector(`#${id}`)?.querySelector('[class*="label"]')?.textContent ?? '';
}

function pathNow(): string {
    return [...document.querySelectorAll('nav[aria-label="Path"] [data-fits]')]
        .map((level) => level.textContent)
        .join(' / ');
}

describe('SpaceCard', () => {
    test('the first segment opens the sections beside the fleet, and one filter narrows both', async () => {
        // Given - linux-1 over two-macs
        let read: string[][] = [];
        await component.render(windowOn(machineRoute('linux-1')), async (visitor) => {
            await visitor.see(LINUX_1);

            // When - Fleet is pressed, then the filter reads lin, sto, then zzz
            await visitor.click(within(path, button('Fleet')));
            await visitor.see(focused(FILTER));
            const open = columns();
            await visitor.fill(FILTER, 'lin');
            const lin = columns();
            await visitor.fill(FILTER, 'sto');
            const sto = columns();
            await visitor.fill(FILTER, 'zzz');
            read = [open, lin, sto, columns()];
        });

        // Then - every section and every machine, then each filter's rows, each empty column saying so
        expect(read).toStrictEqual([
            [
                'Sections: Home, Fleet, Apps, Skills, Machine settings, Machine files, Storage, Passwords, Engine',
                'Fleet: macbook-pro, api, linux-1, mac-mini, infra',
            ],
            ['Sections: No section matches', 'Fleet: linux-1'],
            ['Sections: Storage', 'Fleet: No machine matches'],
            ['Nothing matches'],
        ]);
    });

    test('the arrows walk the sections then the fleet, typing starts again at the first row, Enter goes', async () => {
        // Given - Storage over two-macs, its card open
        let read: string[] = [];
        await component.render(windowOn({ page: 'storage' }), async (visitor) => {
            await visitor.click(within(path, button('Storage')));
            await visitor.see(focused(FILTER));
            const current = activeRow();

            // When - ↓ three times, from Storage past Passwords and Engine into the fleet
            await visitor.press('ArrowDown');
            await visitor.press('ArrowDown');
            await visitor.press('ArrowDown');
            const crossed = activeRow();

            // When - a filter is typed, then ↓ and Enter
            await visitor.fill(FILTER, 'a');
            const typed = activeRow();
            await visitor.press('ArrowDown');
            await visitor.press('Enter');
            await visitor.gone(dialog('Storage'));
            read = [current, crossed, typed, pathNow()];
        });

        // Then - Storage first, macbook-pro after Engine, Apps first of a, then Machine settings opened
        expect(read).toStrictEqual(['Storage', 'macbook-pro', 'Apps', 'Machine settings']);
    });

    test('⌘L opens the card from anywhere on its segment, and Escape returns the keyboard to the segment', async () => {
        // Given - linux-1 over two-macs, the keyboard on nothing
        const held: (null | string)[] = [];
        await component.render(windowOn(machineRoute('linux-1')), async (visitor) => {
            await visitor.see(LINUX_1);

            // When - ⌘L, then Escape
            await visitor.press(COMMAND_L);
            await visitor.see(dialog('Fleet'));
            await visitor.see(focused(FILTER));
            held.push(document.activeElement?.getAttribute('role') ?? null);
            await visitor.press('Escape');
            await visitor.gone(dialog('Fleet'));
            await visitor.see(focused(FLEET));
            held.push(document.activeElement?.textContent ?? null);
        });

        // Then - the card opened on its filter, and closed with the keyboard on Fleet
        expect(held).toStrictEqual(['combobox', 'Fleet']);
    });

    test("the card's Fleet returns to the machine left, and from a window that showed none, to the first Mac", async () => {
        // Given - linux-1's Ports over two-macs
        let read: string[] = [];
        await component.render(windowOn(machineRoute('linux-1', 'Ports')), async (visitor) => {
            await visitor.see(LINUX_1);

            // When - Storage from the card, then the card's Fleet
            await visitor.click(within(path, button('Fleet')));
            await visitor.fill(FILTER, 'Storage');
            await visitor.press('Enter');
            await visitor.see(within(path, button('Storage')));
            await visitor.click(within(path, button('Storage')));
            await visitor.fill(FILTER, 'Fleet');
            await visitor.press('Enter');
            await visitor.see(LINUX_1);
            read = [
                pathNow(),
                document.querySelector('[role="tab"][aria-selected="true"]')?.textContent ?? '',
            ];
        });
        let first = '';
        await component.render(windowOn({ page: 'passwords' }), async (visitor) => {
            // When - the card's Fleet, from a window that opened on Passwords
            await visitor.click(within(path, button('Passwords')));
            await visitor.fill(FILTER, 'Fleet');
            await visitor.press('Enter');
            await visitor.see(within(path, button('macbook-pro', { exact: false })));
            first = pathNow();
        });

        // Then - linux-1 on Ports, then macbook-pro
        expect(read).toStrictEqual(['Fleet / macbook-pro / linux-1', 'Ports']);
        expect(first).toBe('Fleet / macbook-pro');
    });
});
