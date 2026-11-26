import {
    banner,
    button,
    component,
    content,
    dialog,
    disabled,
    focused,
    navigation,
    option,
    waitUntil,
    within,
} from '@jterrazz/test';
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

const bar = banner();
const path = navigation('Path');
const spaces = navigation('Spaces');
const LINUX_1 = within(path, button('linux-1', { exact: false }));

// hazard: press wraps a key longer than one letter in braces, so a chord is user-event's hold and release
// hazard: with its outer braces left to press.
const COMMAND_2 = 'Meta>}2{/Meta';
const COMMAND_1 = 'Meta>}1{/Meta';
const COMMAND_0 = 'Meta>}0{/Meta';

function selected(): string | undefined {
    return document.querySelector('[role="tab"][aria-selected="true"]')?.textContent;
}

function windowOn(initial: Route) {
    return (
        <EngineProvider dependencies={composeReplay({ recording: recordingFor('two-macs') })}>
            <Window initial={initial} memory={PINNED} pages={PAGES} />
        </EngineProvider>
    );
}

function words(): HTMLButtonElement[] {
    return [...document.querySelectorAll<HTMLButtonElement>('nav[aria-label="Spaces"] button')];
}

// value: the words the bar underlines, by their drawn name: one, or none on a page above the spaces.
function pressedWords(): string[] {
    return words()
        .filter((word) => word.getAttribute('aria-pressed') === 'true')
        .map((word) => word.firstElementChild?.textContent ?? '');
}

async function underlined(expected: readonly string[]): Promise<void> {
    await waitUntil(() => pressedWords().join('/') === expected.join('/'), {
        why: `the bar underlines ${expected.join(', ') || 'no space'}`,
    });
}

describe('spaces', () => {
    test('the underline follows the route: Machine on a machine and on Home, none on Settings, back with Back', async () => {
        // Given - the window on linux-1 over two-macs
        let read: string[][] = [];
        await component.render(windowOn(machineRoute('linux-1')), async (visitor) => {
            await visitor.see(LINUX_1);
            const machine = pressedWords();

            // When - Settings, then Back, then ⌘0
            await visitor.click(within(bar, button('Settings')));
            await visitor.see(within(path, content('Settings')));
            await underlined([]);
            const settings = pressedWords();
            await visitor.click(within(bar, button('Back')));
            await visitor.see(LINUX_1);
            await underlined(['Machine']);
            const back = pressedWords();
            await visitor.press(COMMAND_0);
            await visitor.see(within(path, content('Home')));
            read = [machine, settings, back, pressedWords()];
        });

        // Then - Machine, none, Machine, Machine
        expect(read).toStrictEqual([['Machine'], [], ['Machine'], ['Machine']]);
    });

    test('⌘2 opens the Soon list on Agent and moves nothing, and Escape returns to the word', async () => {
        // Given - the window on linux-1 over two-macs, no page before it
        let pressed: string[] = [];
        await component.render(windowOn(machineRoute('linux-1')), async (visitor) => {
            await visitor.see(LINUX_1);

            // When - ⌘2
            await visitor.press(COMMAND_2);

            // Then - the spaces listed, Agent and OS saying soon; linux-1 still open, Machine still underlined
            const list = dialog('Spaces');
            await visitor.see(
                within(list, content('Soon · The workforce: personalities, roles, focus')),
            );
            await visitor.see(LINUX_1);
            await visitor.see(disabled(button('Back')));
            pressed = pressedWords();

            // When - Escape
            await visitor.press('Escape');

            // Then - the list closes, the focus on the word that opened it
            await visitor.gone(list);
            await visitor.see(focused(button('Agent soon')));
        });
        expect(pressed).toStrictEqual(['Machine']);
    });

    test('pressing Agent opens the same list, and pressing it again closes it', async () => {
        // Given - the window on Home over two-macs
        let pressed: string[] = [];
        await component.render(windowOn({ page: 'home' }), async (visitor) => {
            await visitor.see(within(path, content('Home')));

            // When - Agent is pressed twice
            await visitor.click(within(spaces, button('Agent soon')));
            await visitor.see(dialog('Spaces'));
            await visitor.click(within(spaces, button('Agent soon')));

            // Then - the list is gone, Home still open and Machine still underlined
            await visitor.gone(dialog('Spaces'));
            await visitor.see(within(path, content('Home')));
            pressed = pressedWords();
        });
        expect(pressed).toStrictEqual(['Machine']);
    });

    test('the arrows move the focus between the words and press none', async () => {
        // Given - the window on linux-1 over two-macs, the focus on Machine
        let pressed: string[] = [];
        await component.render(windowOn(machineRoute('linux-1')), async (visitor) => {
            await visitor.see(LINUX_1);
            words()[0]?.focus();

            // When - → twice, then ←
            await visitor.press('ArrowRight');
            await visitor.see(focused(button('Agent soon')));
            await visitor.press('ArrowRight');
            await visitor.see(focused(button('OS soon')));
            await visitor.press('ArrowLeft');

            // Then - the focus on Agent, no list open, linux-1 still open and Machine still underlined
            await visitor.see(focused(button('Agent soon')));
            await visitor.gone(dialog('Spaces'));
            await visitor.see(LINUX_1);
            pressed = pressedWords();
        });
        expect(pressed).toStrictEqual(['Machine']);
    });

    test('⌘1 from Home returns to the machine I left, and pressing Machine does the same', async () => {
        // Given - the window on linux-1's Ports over two-macs
        let tabs: (null | string | undefined)[] = [];
        await component.render(windowOn(machineRoute('linux-1', 'Ports')), async (visitor) => {
            await visitor.see(LINUX_1);

            // When - Home by ⌘0, then ⌘1; Home again by the card's first row, then the Machine word
            await visitor.press(COMMAND_0);
            await visitor.see(within(path, content('Home')));
            await visitor.press(COMMAND_1);
            await visitor.see(LINUX_1);
            const byKey = selected();
            await visitor.click(within(path, button('Fleet')));
            await visitor.click(option('Home ⌘0'));
            await visitor.see(within(path, content('Home')));
            await visitor.click(within(spaces, button('Machine')));
            await visitor.see(LINUX_1);
            tabs = [byKey, selected()];
        });

        // Then - linux-1 again both times, on the tab I left
        expect(tabs).toStrictEqual(['Ports', 'Ports']);
    });
});
