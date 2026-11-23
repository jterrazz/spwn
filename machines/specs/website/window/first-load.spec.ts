import { content, main, region, status, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('first load', () => {
    test('A8 a page not answered yet draws its own panel and says it is reading', async () => {
        // Given - the window on the Passwords page over two-macs, vault.list never answered
        const result = await website.visit(
            '/?world=two-macs&page=passwords&unread=vault.list',
            async (visitor) => {
                // When - the page is drawn
                // Then - it says what it reads, and its drawn panel stays hidden from assistive tech until answered
                await visitor.see(within(main(), content('Reading the passwords of this Mac…')));
                await visitor.gone(region('Passwords'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('A9 a refresh refused keeps the page as answered, and the toolbar says what the engine said and since when', async () => {
        // Given - macbook-pro's Ports over stories, its link absent for a moment so its reads are asked again, every ask of engine.read but the first refused
        const result = await website.visit(
            '/?world=stories&machine=macbook-pro&tab=Ports&stale=engine.read&link=blip',
            async (visitor) => {
                const listening = region('What listens on this Mac');
                await visitor.see(within(listening, content('ControlCenter', { exact: false })));

                // When - the next refresh of engine.read is refused
                await visitor.see(status('macbook-air does not answer'));

                // Then - the listing stays drawn as answered, the mark beside the time of that answer
                await visitor.see(content('as of', { exact: false }));
                await visitor.see(within(listening, content('ControlCenter', { exact: false })));
                await visitor.see(within(listening, content('Outside Tailscale')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
