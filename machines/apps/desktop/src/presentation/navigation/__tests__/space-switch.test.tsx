import '../../ui/document.css';
import {
    banner,
    button,
    component,
    content,
    dialog,
    focused,
    navigation,
    waitUntil,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { machineRoute } from '../route';
import { Window } from '../window';
import { PINNED } from './pinned';

const recordingFor = await recordingsOf('two-macs');

const path = navigation('Path');

function one(selector: string): HTMLElement {
    const found = document.querySelector<HTMLElement>(selector);
    if (found === null) {
        throw new Error(`nothing drawn matches ${selector}`);
    }
    return found;
}

function px(value: string): number {
    return Number.parseFloat(value) || 0;
}

function selectedTab(): string {
    return one('[role="tab"][aria-selected="true"]').textContent;
}

// value: the two levels as drawn: the bar and its spaces, the page's tab bar and its tabs.
function levels() {
    const bar = one('header');
    const spaces = one('nav[aria-label="Spaces"]');
    const [machine, agent] = [...spaces.querySelectorAll<HTMLButtonElement>('button')];
    const tablist = one('[role="tablist"]');
    const tab = one('[role="tab"][aria-selected="true"]');
    const restTab = one('[role="tab"][aria-selected="false"]');
    if (machine === undefined || agent === undefined) {
        throw new Error('the spaces draw fewer than two words');
    }
    return { agent, bar, machine, restTab, spaces, tab, tablist };
}

// value: the bottom of the line a word draws under it, its `::after`, in the window's px.
function lineBottomOf(word: HTMLElement): number {
    const line = getComputedStyle(word, '::after');
    return word.getBoundingClientRect().bottom - px(line.bottom);
}

// value: where each word starts and how wide it is, in the window's px, Machine, Agent then OS.
function edges(): string {
    return [...one('nav[aria-label="Spaces"]').querySelectorAll('button')]
        .map((word) => {
            const { left, width } = word.getBoundingClientRect();
            return `${String(Math.round(left * 100) / 100)}+${String(Math.round(width * 100) / 100)}`;
        })
        .join(' ');
}

describe('SpaceSwitch', () => {
    test('a word keeps its width whatever is pressed: on a machine, with Agent listed, and on Settings', async () => {
        // Given - linux-1 over two-macs, the window's tokens loaded
        let read: string[] = [];
        await component.viewport({ height: 900, width: 1440 }).render(
            <EngineProvider dependencies={composeReplay({ recording: recordingFor('two-macs') })}>
                <Window initial={machineRoute('linux-1')} memory={PINNED} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(within(path, button('linux-1', { exact: false })));
                await document.fonts.ready;
                const onMachine = edges();

                // When - Agent's list opens, then Settings
                await visitor.click(button('Agent soon'));
                await visitor.see(dialog('Spaces'));
                const listed = edges();
                await visitor.press('Escape');
                await visitor.gone(dialog('Spaces'));
                await visitor.click(within(banner(), button('Settings')));
                await visitor.see(within(path, content('Settings')));
                await waitUntil(
                    () =>
                        document.querySelector('nav[aria-label="Spaces"] [aria-pressed="true"]') ===
                        null,
                    { why: 'no word pressed on Settings' },
                );
                read = [onMachine, listed, edges()];
            },
        );

        // Then - the three words start and end where they did
        expect(read[1]).toBe(read[0]);
        expect(read[2]).toBe(read[0]);
    });

    test('the spaces and a page tab read apart: the line, the weight, the corner, the air and the roles', async () => {
        // Given - linux-1's Ports over two-macs, 1440 px wide, the window's tokens loaded
        let measured: Record<string, boolean | number | string> = {};
        await component.viewport({ height: 900, width: 1440 }).render(
            <EngineProvider dependencies={composeReplay({ recording: recordingFor('two-macs') })}>
                <Window initial={machineRoute('linux-1', 'Ports')} memory={PINNED} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(within(path, button('linux-1', { exact: false })));
                await waitUntil(() => document.querySelector('[role="tablist"]') !== null, {
                    why: "linux-1's tabs drawn",
                });

                // When - both levels are measured
                const { agent, bar, machine, restTab, spaces, tab, tablist } = levels();
                const barBottom = bar.getBoundingClientRect().bottom;
                const rule = spaces.nextElementSibling;
                const ruleLeft = rule?.getBoundingClientRect().left ?? 0;
                const word = getComputedStyle(machine);
                measured = {
                    activeOutweighsTab: px(word.fontWeight) > px(getComputedStyle(tab).fontWeight),
                    airBetweenWords: Math.round(
                        agent.getBoundingClientRect().left - machine.getBoundingClientRect().right,
                    ),
                    lineHeight: px(getComputedStyle(machine, '::after').height),
                    lineOnBarFoot: Math.abs(lineBottomOf(machine) - barBottom) <= 1,
                    restOutweighsTab:
                        px(getComputedStyle(agent).fontWeight) >
                        px(getComputedStyle(restTab).fontWeight),
                    ruleClosesSpaces:
                        ruleLeft >= spaces.getBoundingClientRect().right &&
                        rule?.getAttribute('aria-hidden') === 'true',
                    tabBelowBar: tab.getBoundingClientRect().bottom - barBottom > 48,
                    tabIsPadded: px(getComputedStyle(tab).paddingLeft) > 0,
                    tabsOutsideBar: !bar.contains(tablist),
                    tabsInSpaces: spaces.querySelectorAll('[role="tab"], [role="tablist"]').length,
                    wordFill: word.backgroundColor,
                    wordIsBare: px(word.paddingLeft) + px(word.paddingRight),
                    wordRole: `${String(machine.getAttribute('role'))} ${String(machine.getAttribute('aria-pressed'))}`,
                };

                // When - → on the spaces, then → on the tabs
                machine.focus();
                await visitor.press('ArrowRight');
                await visitor.see(focused(button('Agent soon')));
                measured.spaceArrowKeeps = `${String(machine.getAttribute('aria-pressed'))} ${selectedTab()}`;
                tab.focus();
                await visitor.press('ArrowRight');
                await waitUntil(() => selectedTab() !== 'Ports', { why: 'the next tab selected' });
                measured.tabArrowSelects = selectedTab();
            },
        );

        // Then - the five differences, as measured
        expect(measured).toStrictEqual({
            activeOutweighsTab: true,
            airBetweenWords: 18,
            lineHeight: 2,
            lineOnBarFoot: true,
            restOutweighsTab: true,
            ruleClosesSpaces: true,
            spaceArrowKeeps: 'true Ports',
            tabArrowSelects: 'Sign-ins',
            tabBelowBar: true,
            tabIsPadded: true,
            tabsInSpaces: 0,
            tabsOutsideBar: true,
            wordFill: 'rgba(0, 0, 0, 0)',
            wordIsBare: 0,
            wordRole: 'null true',
        });
    });
});
