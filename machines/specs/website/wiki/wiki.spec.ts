import { button, content, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const rail = navigation('Rail');
const seats = navigation('Seats');

describe('wiki', () => {
    test('O1 the rail’s OS opens the wiki’s home, its pages under the seat that reads them', async () => {
        // Given - the window on Home over os-wiki
        const result = await website.visit('/?world=os-wiki&page=home', async (visitor) => {
            await visitor.see(within(rail, button('OS')));

            // When - I press OS in the rail
            await visitor.click(within(rail, button('OS')));

            // Then - the wiki's home, a section for each seat that holds a page
            await visitor.see(heading('Wiki'));
            await visitor.see(within(region('Vision'), content('Philosophy')));
            await visitor.see(within(region('Produit'), content('Boundaries')));
            await visitor.see(within(region('Technique'), content('Ports')));
            await visitor.see(within(region('Croissance'), content('Positioning')));
        });
        expect(result.errors.text).toBe('');
    });

    test('O2 a page’s tile opens the page, its seat chosen, at the reading time the engine chose', async () => {
        // Given - the wiki's home over os-wiki
        const result = await website.visit('/?world=os-wiki&page=wiki', async (visitor) => {
            await visitor.see(heading('Wiki'));

            // When - I press the tile of Boundaries
            await visitor.click(within(region('Produit'), button('Boundaries', { exact: false })));

            // Then - the page, Produit chosen, at two minutes
            await visitor.see(heading('Boundaries'));
            await visitor.see(content('E1 Produit'));
        });
        expect(result.errors.text).toBe('');
    });

    test('O3 ten minutes picked serves the page again, its counted blocks shown', async () => {
        // Given - the ports page at two minutes, its quiet block counted
        const result = await website.visit(
            '/?world=os-wiki&page=wiki&item=machines/reference/ports',
            async (visitor) => {
                await visitor.see(content('1 quiet block'));

                // When - I pick 10min
                await visitor.click(button('10min'));

                // Then - the quiet block is a line of its own, and nothing is counted any more
                await visitor.see(content('audience'));
                await visitor.gone(content('1 quiet block'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('O4 a seat opens its home page, that seat chosen', async () => {
        // Given - the Boundaries page over os-wiki
        const result = await website.visit(
            '/?world=os-wiki&page=wiki&item=product/boundaries',
            async (visitor) => {
                await visitor.see(heading('Boundaries'));

                // When - I press Technique
                await visitor.click(within(seats, button('Technique')));

                // Then - Ports, Technique chosen
                await visitor.see(heading('Ports'));
                await visitor.see(content('E3 Architecture'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
