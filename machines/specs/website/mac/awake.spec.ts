import { button, content, dialog, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('awake', () => {
    test('M2 Awake keeps the Mac awake for 4 h, and Home says until when', async () => {
        // Given - This Mac's page over two-macs, whose awake.read answers three choices and no session
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&pace=600',
            async (visitor) => {
                // When - Awake… is pressed, then 4 h in its sheet
                await visitor.click(button('Keep macbook-pro awake'));
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.see(within(sheet, button('4 h')));
                await visitor.gone(within(sheet, button('Stop keeping awake')));
                await visitor.click(within(sheet, button('4 h')));

                // Then - the sheet closes, and Home's Awake tile says On, until when
                await visitor.gone(sheet);
                await visitor.press('Meta+0');
                await visitor.see(content('on'));
                await visitor.see(content('until', { exact: false }));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M2 a Mac kept awake offers Stop keeping awake, and Home turns the tile off', async () => {
        // Given - This Mac's page over stories, whose awake.read answers a session
        const result = await website.visit(
            '/?world=stories&machine=macbook-pro&pace=600',
            async (visitor) => {
                // When - Awake… is pressed, then Stop keeping awake in its sheet
                await visitor.click(button('Keep macbook-pro awake'));
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.click(within(sheet, button('Stop keeping awake')));

                // Then - the sheet closes, and Home's Awake tile no longer says On
                await visitor.gone(sheet);
                await visitor.press('Meta+0');
                await visitor.see(content('off'));
                await visitor.gone(content('on'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M14 the Awake sheet says until when the Mac is kept awake, above the terms to choose from', async () => {
        // Given - This Mac's page over kept-four-hours, whose engine keeps it awake for the 4h asked, until 14:00
        const result = await website.visit(
            '/?world=kept-four-hours&machine=macbook-pro',
            async (visitor) => {
                // When - Awake… is pressed
                await visitor.click(button('Keep macbook-pro awake'));

                // Then - the sheet says until when, then offers the terms, and Stop keeping awake
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.see(within(sheet, content('Awake until 14:00.')));
                await visitor.see(within(sheet, button('4 h')));
                await visitor.see(within(sheet, button('Stop keeping awake')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
