import { banner, button, content, dialog, heading, navigation, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const SAID = within(sidebar, content('No engine answers at replay of two-macs'));

describe('link', () => {
    test('E7 an engine absent for a moment leaves the page drawn and its sheet open', async () => {
        // Given - Home over two-macs, whose link is declared absent after a second, for three
        const result = await website.visit(
            '/?world=two-macs&page=home&link=blip',
            async (visitor) => {
                await visitor.see(heading('2 things need you.'));
                await visitor.click(within(banner(), button('Find')));

                // When - the engine goes, then answers again
                await visitor.see(SAID);
                await visitor.gone(SAID);

                // Then - Home is still drawn under the open Find, and no panel replaced it
                await visitor.see(dialog('Find'));
                await visitor.see(heading('2 things need you.'));
                await visitor.gone(heading('Spwn is not installed on this Mac.'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
