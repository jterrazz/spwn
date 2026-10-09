import { button, dialog } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

// invariant: the shortcut is bound by an effect, so a visitor reaches for it only once the client has read the Docker daemon
const settled = button('Docker status: v', { exact: false });

describe('palette', () => {
    test('opens on the command shortcut', async () => {
        // Given - the landing page, hydrated
        const result = await website.visit('/', async (visitor) => {
            await visitor.see(settled);

            // When - the visitor presses the shortcut
            await visitor.press('Meta+k');
            await visitor.see(dialog('Command Palette'));
        });

        // Then - the palette offers to search the commands
        expect(result.content.text).toContain('Search for a command to run...');
    });

    test('closes on escape', async () => {
        // Given - the palette, open
        const result = await website.visit('/', async (visitor) => {
            await visitor.see(settled);
            await visitor.press('Meta+k');
            await visitor.see(dialog('Command Palette'));

            // When - the visitor dismisses it
            await visitor.press('Escape');
            await visitor.gone(dialog('Command Palette'));
        });

        // Then - nothing of the palette stays on the page
        expect(result.content.text).not.toContain('Search for a command to run...');
    });
});
