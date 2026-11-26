import {
    banner,
    button,
    component,
    content,
    dialog,
    field,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { Profiler } from 'react';
import { describe, expect, test } from 'vitest';

import { settled } from '../../../composition/__tests__/page-over';
import { composeReplay } from '../../../composition/compose';
import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { CHANGES, opened } from '../../dependencies/__tests__/over-a-replay';
import { pushed } from '../../dependencies/__tests__/pushed';
import { EngineProvider } from '../../dependencies/dependencies';
import { HIDDEN, PINNED } from '../../navigation/__tests__/pinned';
import type { Route } from '../../navigation/route';
import type { Memory } from '../../navigation/route-memory';
import { Window } from '../../navigation/window';
import { PAGES } from '../registry';
import { overBudget, tallied } from './tally';
import type { Budget, Tally } from './tally';

const recordingFor = await recordingsOf('calm', 'create-held', 'stories');

// hazard: press wraps a key longer than one letter in braces, so a chord is user-event's hold and release
// hazard: with its outer braces left to press.
const COMMAND_K = 'Control>}k{/Control';

const HOME: Route = { page: 'home' };

// value: the most React commits and component renders a gesture may cost, from its first render to the page settled,
// value: each the count measured when it was last lowered: a change that costs more fails here, one that costs less
// value: lowers it.
const BUDGETS = {
    activityTab: { commits: 4, renders: 326 },
    find: { commits: 1, renders: 25 },
    guideOpens: { commits: 4, renders: 294 },
    homeCalm: { commits: 8, renders: 543 },
    homeStories: { commits: 8, renders: 801 },
    newMachine: { commits: 2, renders: 39 },
    peek: { commits: 1, renders: 75 },
    pin: { commits: 1, renders: 74 },
    storage: { commits: 2, renders: 417 },
    unchangedRevision: { commits: 0, renders: 0 },
} as const satisfies Record<string, Budget>;

// hazard: the harness loads no tokens, so the column's edge has no height to hover: the pointer event is sent by hand.
function pointerEnters(element: Element | null): void {
    element?.dispatchEvent(new PointerEvent('pointerover', { bubbles: true }));
}

const UNDRAWN = { Body: 0, HomePage: 0, Main: 0, PageArea: 0 };

// value: the renders of the components that draw the page beside the column: a gesture of the column renders none.
function pageRendersOf(cost: Tally | undefined): typeof UNDRAWN {
    const by = cost?.by ?? {};
    return {
        Body: by.Body ?? 0,
        HomePage: by.HomePage ?? 0,
        Main: by.Main ?? 0,
        PageArea: by.PageArea ?? 0,
    };
}

type Visitor = Parameters<NonNullable<Parameters<typeof component.render>[1]>>[0];

// value: the window over `world` with the pages the registry hands it, open from the start, its column as `memory`
// value: keeps it, its engine.state pushed by the spec and every React commit and component render tallied.
function talliedOver(world: string, initial: Route, memory: Memory = PINNED) {
    const stage = opened(recordingFor(world));
    const engine = pushed(stage.transport);
    const tally = tallied();
    const ui = (
        <EngineProvider
            dependencies={{
                ...composeReplay({ recording: recordingFor(world) }),
                engine: new Client(engine.transport),
            }}
        >
            <Profiler id="window" onRender={tally.onRender}>
                <Window initial={initial} memory={memory} pages={PAGES} />
            </Profiler>
        </EngineProvider>
    );
    return { engine, stage, tally, ui };
}

type Tallied = ReturnType<typeof talliedOver>;

// value: what `gesture` costs on a window drawn whole: counted from the gesture to every read it asked answered.
async function costOf(over: Tallied, gesture: (visitor: Visitor) => Promise<void>) {
    let cost: Tally | undefined;
    await component.render(over.ui, async (visitor) => {
        await settled(over.stage);
        over.tally.taken();
        await gesture(visitor);
        await settled(over.stage);
        cost = over.tally.taken();
    });
    return cost;
}

describe('registry', () => {
    test.each([
        ['calm', BUDGETS.homeCalm],
        ['stories', BUDGETS.homeStories],
    ])(
        'home over %s is drawn within its budget of commits and renders, from its first render to every read answered',
        async (world, budget) => {
            // Given - the window on Home over the world, its commits and renders counted from its first render
            const over = talliedOver(world, HOME);
            let load: Tally | undefined;

            // When - every read Home asks has answered and the page is drawn
            await component.render(over.ui, async () => {
                await settled(over.stage);
                load = over.tally.taken();
            });

            // Then - it cost no more than its budget
            expect(overBudget(load, budget)).toStrictEqual({});
        },
    );

    test('home to Storage is drawn within its budget of commits and renders', async () => {
        // Given - the window on Home over stories, drawn whole
        const over = talliedOver('stories', HOME);

        // When - the column's Storage is clicked
        const cost = await costOf(over, async (visitor) => {
            await visitor.click(within(navigation('Column'), button('Storage')));
        });

        // Then - it cost no more than its budget
        expect(overBudget(cost, BUDGETS.storage)).toStrictEqual({});
    });

    test('the New machine sheet opens within its budget of commits and renders', async () => {
        // Given - the window on Home over stories, drawn whole
        const over = talliedOver('stories', HOME);

        // When - the bar's New machine is clicked
        const cost = await costOf(over, async (visitor) => {
            await visitor.click(within(banner(), button('New machine')));
        });

        // Then - it cost no more than its budget
        expect(overBudget(cost, BUDGETS.newMachine)).toStrictEqual({});
    });

    test('find opens within its budget of commits and renders', async () => {
        // Given - the window on Home over stories, drawn whole
        const over = talliedOver('stories', HOME);

        // When - Find is opened from the keyboard
        const cost = await costOf(over, async (visitor) => {
            await visitor.press(COMMAND_K);
        });

        // Then - it cost no more than its budget
        expect(overBudget(cost, BUDGETS.find)).toStrictEqual({});
    });

    test("a Mac's Activity tab opens within its budget of commits and renders", async () => {
        // Given - mac-mini's Overview over stories, drawn whole
        const over = talliedOver('stories', {
            machine: 'mac-mini',
            page: 'machine',
            tab: 'Overview',
        });

        // When - its Activity tab is clicked
        const cost = await costOf(over, async (visitor) => {
            await visitor.click(content('Activity'));
        });

        // Then - it cost no more than its budget
        expect(overBudget(cost, BUDGETS.activityTab)).toStrictEqual({});
    });

    test('launching a machine hands New machine to its Guide in one commit, within its budget', async () => {
        // Given - the New machine sheet filled for linux-2 over create-held, every moment of the window's sheets seen
        const over = talliedOver('create-held', HOME);
        const sheet = dialog('New machine on macbook-pro');
        const bare: string[] = [];
        const watcher = new MutationObserver(() => {
            if (document.querySelector('[role="dialog"]') === null) {
                bare.push(document.body.textContent);
            }
        });
        let cost: Tally | undefined;
        await component.render(over.ui, async (visitor) => {
            await settled(over.stage);
            await visitor.click(within(banner(), button('New machine')));
            await visitor.click(within(region('On'), button('macbook-pro')));
            await visitor.fill(within(sheet, field('What is it for?')), 'Lot shell');
            await visitor.fill(within(sheet, field('Name')), 'linux-2');
            await visitor.click(within(sheet, button('ephemeral')));
            await settled(over.stage);
            over.tally.taken();
            watcher.observe(document.body, CHANGES);

            // When - Launch is pressed and the engine begins the act
            await visitor.click(within(sheet, button('Launch')));
            await visitor.see(dialog('New machine linux-2'));
            await settled(over.stage);
            cost = over.tally.taken();
            watcher.disconnect();
        });

        // Then - no moment drew the window without a sheet, and it cost no more than its budget
        expect(bare).toStrictEqual([]);
        expect(overBudget(cost, BUDGETS.guideOpens)).toStrictEqual({});
    });

    test('a revision of engine.state equal to the one drawn commits nothing and renders nothing', async () => {
        // Given - the window on Home over stories, drawn whole
        const over = talliedOver('stories', HOME);

        // When - the engine pushes a revision whose views equal those drawn
        const cost = await costOf(over, async () => {
            over.engine.same();
            await Promise.resolve();
        });

        // Then - not one commit, not one render
        expect(overBudget(cost, BUDGETS.unchangedRevision)).toStrictEqual({});
    });

    test('pinning the column draws the column and the pin within their budget, and never the page beside it', async () => {
        // Given - the window on Home over stories, its column hidden, drawn whole
        const over = talliedOver('stories', HOME, HIDDEN);

        // When - the bar's Pin the column is pressed
        const cost = await costOf(over, async (visitor) => {
            await visitor.click(within(banner(), button('Pin the column')));
            await visitor.see(navigation('Column'));
        });

        // Then - it cost no more than its budget, and the page rendered not once
        expect(overBudget(cost, BUDGETS.pin)).toStrictEqual({});
        expect(pageRendersOf(cost)).toStrictEqual(UNDRAWN);
    });

    test('peeking the column draws the column within its budget, and never the page beneath it', async () => {
        // Given - the window on Home over stories, its column hidden, drawn whole
        const over = talliedOver('stories', HOME, HIDDEN);

        // When - the pointer reaches the window's left edge
        const cost = await costOf(over, async (visitor) => {
            pointerEnters(document.querySelector('[data-testid="column-edge"]'));
            await visitor.see(navigation('Column'));
        });

        // Then - it cost no more than its budget, and the page rendered not once
        expect(overBudget(cost, BUDGETS.peek)).toStrictEqual({});
        expect(pageRendersOf(cost)).toStrictEqual(UNDRAWN);
    });
});
