import { button, component, content, main, navigation, within } from '@jterrazz/test';
import { Profiler } from 'react';
import { describe, expect, test } from 'vitest';

import { settled } from '../../../composition/__tests__/page-over';
import { composeReplay } from '../../../composition/compose';
import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { opened } from '../../dependencies/__tests__/over-a-replay';
import { pushed } from '../../dependencies/__tests__/pushed';
import { EngineProvider } from '../../dependencies/dependencies';
import type { Route } from '../../navigation/route';
import { Window } from '../../navigation/window';
import { PAGES } from '../registry';
import { overBudget, tallied } from './tally';
import type { Budget, Tally } from './tally';

const recordingFor = await recordingsOf('calm', 'stories');

// hazard: press wraps a key longer than one letter in braces, so a chord is user-event's hold and release
// hazard: with its outer braces left to press.
const COMMAND_K = 'Control>}k{/Control';

const HOME: Route = { page: 'home' };

// value: the most React commits and component renders a gesture may cost, from its first render to the page settled,
// value: each the count measured when it was last lowered: a change that costs more fails here, one that costs less
// value: lowers it.
const BUDGETS = {
    activityTab: { commits: 4, renders: 318 },
    find: { commits: 1, renders: 265 },
    homeCalm: { commits: 8, renders: 570 },
    homeStories: { commits: 8, renders: 815 },
    newMachine: { commits: 2, renders: 279 },
    storage: { commits: 2, renders: 412 },
    unchangedRevision: { commits: 0, renders: 0 },
} as const satisfies Record<string, Budget>;

type Visitor = Parameters<NonNullable<Parameters<typeof component.render>[1]>>[0];

// value: the window over `world` with the pages the registry hands it, open from the start, its engine.state pushed
// value: by the spec and every React commit and component render tallied.
function talliedOver(world: string, initial: Route) {
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
                <Window initial={initial} pages={PAGES} />
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

        // When - the sidebar's Storage is clicked
        const cost = await costOf(over, async (visitor) => {
            await visitor.click(within(navigation('Sidebar'), button('Storage')));
        });

        // Then - it cost no more than its budget
        expect(overBudget(cost, BUDGETS.storage)).toStrictEqual({});
    });

    test('the New machine sheet opens within its budget of commits and renders', async () => {
        // Given - the window on Home over stories, drawn whole
        const over = talliedOver('stories', HOME);

        // When - Home's New machine is clicked
        const cost = await costOf(over, async (visitor) => {
            await visitor.click(within(main(), button('New machine')));
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
});
