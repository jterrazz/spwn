import { button, component, content, file, heading, region, within } from '@jterrazz/test';
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
