import { button, heading, navigation, testId, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { PINNED, website } from '../website.specification.js';

const PORTS = '/?world=two-macs&machine=linux-1&tab=Ports';
const column = navigation('Column');
const edge = testId('column-edge');
const pin = button('Pin the column');

describe('column', () => {
    test('A20 below 1440 px the column hides, and the left edge peeks it until a machine is chosen', async () => {
        // Given - linux-1's Ports over two-macs in a 1280 px window that never pinned its column
        const result = await website.visit(PORTS, async (visitor) => {
            await visitor.see(heading('linux-1 running'));
            await visitor.gone(column);

            // When - the pointer reaches the window's left edge, then api is chosen in the column
            await visitor.hover(edge);
            await visitor.see(within(column, button('macbook-pro online')));
            await visitor.click(within(column, button('api stopped')));

            // Then - api opens, and the peek is gone with the choice
            await visitor.see(heading('api stopped'));
            await visitor.gone(column);
        });
        expect(result.errors.text).toBe('');
    });

    test('A21 the peek pins the column, and the next launch opens it pinned', async () => {
        // Given - linux-1's Ports over two-macs, the column hidden at 1280 px
        const result = await website.visit(PORTS, async (visitor) => {
            // When - the column is peeked and its foot's Pin the column pressed, then the window launched again
            await visitor.hover(edge);
            await visitor.click(within(column, pin));
            await visitor.gone(edge);
            await visitor.goto(PORTS);

            // Then - the column stands beside the page with no edge to peek it, its Library pages there
            await visitor.see(heading('linux-1 running'));
            await visitor.see(within(column, button('Passwords')));
            await visitor.gone(edge);
        });
        expect(result.errors.text).toBe('');
    });

    test('A22 ⌃⌘S unpins a pinned column and pins it again', async () => {
        // Given - linux-1's Ports over two-macs, the column pinned
        const result = await website.storage(PINNED).visit(PORTS, async (visitor) => {
            await visitor.see(within(column, button('macbook-pro online')));

            // When - ⌃⌘S is pressed
            await visitor.press('Control+Meta+KeyS');

            // Then - the column is gone, and ⌃⌘S brings it back
            await visitor.gone(column);
            await visitor.press('Control+Meta+KeyS');
            await visitor.see(within(column, button('macbook-pro online')));
        });
        expect(result.errors.text).toBe('');
    });
});
