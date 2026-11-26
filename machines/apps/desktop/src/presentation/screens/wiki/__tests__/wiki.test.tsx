import { button, component, content, file, focused, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('os-wiki');

const SOURCES = import.meta.glob<string>('../*.{ts,tsx}', {
    eager: true,
    import: 'default',
    query: '?raw',
});

function wikiOver(item?: string) {
    return pageOver(
        recordingFor('os-wiki'),
        item === undefined ? { page: 'wiki' } : { item, page: 'wiki' },
    );
}

const PAGES = [
    { item: 'blueprint/philosophy', golden: 'philosophy', title: 'Philosophy' },
    { item: 'product/stack', golden: 'stack', title: 'The stack' },
    { item: 'product/boundaries', golden: 'boundaries', title: 'Boundaries' },
    { item: 'growth/positioning', golden: 'positioning', title: 'Positioning' },
    { item: 'machines/reference/ports', golden: 'ports', title: 'Ports' },
] as const;

const QUIET_LINE =
    'to be a second place where facts live, or to hide the detail with no way down to it os-holds-no-facts';

// value: more stops than the window and the page hold before the reading time.
const TABS = 60;

function onTenMinutes(): boolean {
    return document.activeElement?.textContent === '10min';
}

describe('Wiki', () => {
    test('the wiki home lists every page under the seat that reads it', async () => {
        // Given - the wiki of os-wiki, opened with no page named
        const result = await component.render(wikiOver(), async (visitor) => {
            // When - the engine has served it
            await visitor.see(heading('Wiki'));
            await visitor.see(within(region('Produit'), content('Boundaries')));
        });

        // Then - All is the chosen seat, and each seat holds its pages as tiles
        await expect(result.tree).toMatch(file('./__expected__/home.aria.yaml'));
    });

    test.each(PAGES)(
        'the $title page draws what the engine served at its own budget',
        async ({ item, golden, title }) => {
            // Given - one page of the wiki of os-wiki
            const result = await component.render(wikiOver(item), async (visitor) => {
                // When - the engine has served it
                await visitor.see(heading(title));
            });

            // Then - its seat chosen, its floor, its face, its sections, groups and blocks as they were served
            await expect(result.tree).toMatch(file(`./__expected__/${golden}.aria.yaml`));
        },
    );

    test('a budget picked asks the page again at that budget', async () => {
        // Given - the ports page at its own budget
        const result = await component.render(
            wikiOver('machines/reference/ports'),
            async (visitor) => {
                await visitor.see(heading('Ports'));

                // When - ten minutes is picked
                await visitor.click(button('10min'));
                await visitor.see(content('A port is a right with an audience.'));
            },
        );

        // Then - the page as the engine serves it for ten minutes
        await expect(result.tree).toMatch(file('./__expected__/ports-10min.aria.yaml'));
    });

    test('a reading time reached by the keyboard asks the page again as a click does', async () => {
        // Given - the ports page at its own budget
        const result = await component.render(
            wikiOver('machines/reference/ports'),
            async (visitor) => {
                await visitor.see(heading('Ports'));

                // When - the keyboard walks to ten minutes and presses it
                for (let step = 0; step < TABS && !onTenMinutes(); step += 1) {
                    await visitor.press('Tab');
                }
                await visitor.see(focused(button('10min')));
                await visitor.press('Enter');
                await visitor.see(content('A port is a right with an audience.'));
            },
        );

        // Then - the very page the click is served
        await expect(result.tree).toMatch(file('./__expected__/ports-10min.aria.yaml'));
    });

    test('a folded group opened draws its blocks whole', async () => {
        // Given - Boundaries at its own budget, its Machines group folded for its seat
        const result = await component.render(wikiOver('product/boundaries'), async (visitor) => {
            await visitor.see(heading('Boundaries'));

            // When - the group is opened
            await visitor.click(within(region('What Machines refuses'), button('Open')));
            await visitor.see(within(region('What Machines refuses'), button('Fold')));
        });

        // Then - the group holds its blocks as the engine served them, and folds back
        await expect(result.tree).toMatch(file('./__expected__/boundaries-group-open.aria.yaml'));
    });

    test('a finding on a block this reading time leaves out opens the reading time that draws it', async () => {
        // Given - Boundaries at thirty seconds, which leaves the OS's refusal out
        const result = await component.render(wikiOver('product/boundaries'), async (visitor) => {
            await visitor.see(heading('Boundaries'));
            await visitor.click(button('30s'));
            await visitor.see(button('needs you os-holds-no-facts shown at 2min'));

            // When - its row in the side panel is pressed
            await visitor.click(button('needs you os-holds-no-facts shown at 2min'));
            await visitor.see(button(QUIET_LINE));
        });

        // Then - the page is the one served at two minutes
        await expect(result.tree).toMatch(file('./__expected__/boundaries.aria.yaml'));
    });

    test('a quiet line opened draws its fields under it', async () => {
        // Given - Boundaries at its own budget, the OS's refusal served as a quiet line
        const result = await component.render(wikiOver('product/boundaries'), async (visitor) => {
            await visitor.see(heading('Boundaries'));

            // When - the line is opened
            await visitor.click(button(QUIET_LINE));
        });

        // Then - the line reads expanded, its fields under it
        await expect(result.tree).toMatch(file('./__expected__/boundaries-line-open.aria.yaml'));
    });

    test('the window reads no tier, score or cause of a block: the engine decided how each shows', () => {
        // Given - the sources of the wiki screen
        const sources = Object.entries(SOURCES);

        // When - each is searched for a block's weight fields
        const weighing = sources
            .filter(([, text]) => /\.(?:Tier|Score|Why)\b/u.test(text))
            .map(([name]) => name);

        // Then - none reads them, and the screen has sources to search
        expect(sources.length).toBeGreaterThan(5);
        expect(weighing).toStrictEqual([]);
    });
});
