import { button, component, file, focused } from '@jterrazz/test';
import { describe, expect, test, vi } from 'vitest';

import { tallied } from '../../../screens/__tests__/tally';
import type { Tally } from '../../../screens/__tests__/tally';
import { drawnRects, Host, reducedMotion, settled, wikiLayout } from './canvas-host';

const PRODUCT = 'Produit, 4 wait for you';

const BOUNDARIES = 'Boundaries, 3 wait for you';

const MAC = 'MacBook Pro, 3 wait for you';

const MACHINE = 'macbook-pro, 2 wait for you';

const LEFT_BEHIND = 'mac-mini, 1 waits for you';

const STATES = [
    { golden: './__expected__/canvas-root.aria.yaml', name: 'the root', path: [] },
    { golden: './__expected__/canvas-seat.aria.yaml', name: 'a seat open', path: ['S:product'] },
    {
        golden: './__expected__/canvas-page.aria.yaml',
        name: 'a page open',
        path: ['S:product', 'P:product/boundaries'],
    },
    {
        golden: './__expected__/canvas-section.aria.yaml',
        name: 'a section open',
        path: ['S:product', 'P:product/boundaries', 'C:product/boundaries:1'],
    },
] as const;

function rectsOf(path: readonly string[]) {
    return new Map(
        [...wikiLayout(path).rects].map(([id, { h, w, x, y }]) => [
            id,
            { h: Math.round(h), w: Math.round(w), x: Math.round(x), y: Math.round(y) },
        ]),
    );
}

describe('Canvas', () => {
    test.each(STATES)(
        '$name: every node is a button named with what waits, and the DOM is the layout',
        async (state) => {
            // Given - the wiki opened down the state's path, under reduced motion
            const restore = reducedMotion(true);
            let drawn = new Map<string, unknown>();
            const result = await component.render(
                <Host sample="wiki" start={state.path} />,
                async (visitor) => {
                    // When - it has drawn
                    await visitor.see(button(PRODUCT));
                    await settled();
                    drawn = drawnRects();
                },
            );
            restore();

            // Then - its tree is the golden, and each node stands where the pure layout put it
            await expect(result.tree).toMatch(file(state.golden));
            expect(drawn).toStrictEqual(rectsOf(state.path));
            expect(result.errors.text).toBe('');
        },
    );

    test('with motion, the FLIP lands every node on its layout rect, and the opened node keeps the focus', async () => {
        // Given - the wiki at its root, with motion
        const restore = reducedMotion(false);
        let drawn = new Map<string, unknown>();
        await component.render(<Host sample="wiki" />, async (visitor) => {
            // When - a seat is opened by a click, then a page by the arrows and Enter
            await visitor.click(button(PRODUCT));
            await visitor.see(focused(button(PRODUCT)));
            await settled();
            await visitor.press('ArrowDown');
            await visitor.see(focused(button(BOUNDARIES)));
            await visitor.press('Enter');

            // Then - the page holds the focus through the move
            await visitor.see(focused(button(BOUNDARIES)));
            await settled();
            drawn = drawnRects();
        });
        restore();

        // Then - every node landed on the layout's rect
        expect(drawn).toStrictEqual(rectsOf(['S:product', 'P:product/boundaries']));
    });

    test("a click on an open node's head closes it to its parent, and Escape goes up one more", async () => {
        // Given - the wiki with a page open
        const onPath = vi.fn<(path: readonly string[]) => void>();
        await component.render(
            <Host onPath={onPath} sample="wiki" start={['S:product', 'P:product/boundaries']} />,
            async (visitor) => {
                // When - the open page's head is clicked, then Escape is pressed
                await visitor.click(button(BOUNDARIES));
                await visitor.see(focused(button(PRODUCT)));
                await visitor.press('Escape');
            },
        );

        // Then - the click closed the page, and Escape the seat
        expect(onPath.mock.calls).toStrictEqual([[['S:product']], [[]]]);
    });

    test('a drag pans the camera without opening what it started on', async () => {
        // Given - the wiki at its root
        const onPath = vi.fn<(path: readonly string[]) => void>();
        let before = 0;
        let after = 0;
        await component.render(<Host onPath={onPath} sample="wiki" />, async (visitor) => {
            await visitor.see(button(PRODUCT));
            before = document.querySelector('[data-node]')?.getBoundingClientRect().left ?? 0;

            // When - a press on a seat drags left and down
            await visitor.drag(button(PRODUCT), { x: 0.5, y: 0.5 }, { x: 0.3, y: 1.5 });
            after = document.querySelector('[data-node]')?.getBoundingClientRect().left ?? 0;
        });

        // Then - the world moved and nothing opened
        expect(after).not.toBe(before);
        expect(onPath).not.toHaveBeenCalled();
    });

    test('the arrows walk the nodes by their places, and the one tab stop follows', async () => {
        // Given - the wiki at its root
        const seen: (Element | null)[] = [];
        let stops = 0;
        await component.render(<Host sample="wiki" />, async (visitor) => {
            await visitor.see(button(PRODUCT));

            // When - Tab enters it, then right, then left
            await visitor.press('Tab');
            seen.push(document.activeElement);
            await visitor.press('ArrowRight');
            seen.push(document.activeElement);
            await visitor.press('ArrowLeft');
            seen.push(document.activeElement);
            stops = document.querySelectorAll('button[data-n][tabindex="0"]').length;
        });

        // Then - the focus went to another node and came back, and only it can be tabbed to
        expect(seen[1]).not.toBe(seen[0]);
        expect(seen[2]).toBe(seen[0]);
        expect(stops).toBe(1);
    });

    test('a second click on the leaf and Space on it hand it to the reader', async () => {
        // Given - the wiki with a section open
        const onRaise = vi.fn<(id: string) => void>();
        await component.render(
            <Host
                onRaise={onRaise}
                sample="wiki"
                start={['S:product', 'P:product/boundaries', 'C:product/boundaries:1']}
            />,
            async (visitor) => {
                // When - a block is clicked twice, then Space is pressed on it
                await visitor.click(button('not-a-method'));
                await visitor.click(button('not-a-method'));
                await visitor.press(' ');
            },
        );

        // Then - the second click and Space raised it
        expect(onRaise.mock.calls).toStrictEqual([
            ['B:product/boundaries:not-a-method'],
            ['B:product/boundaries:not-a-method'],
        ]);
    });

    test('opening a node renders the canvas and only the nodes whose face changed, born or left', async () => {
        // Given - the fleet at its root, its renders counted
        const tally = tallied();
        const taken: Tally[] = [];
        await component.render(
            <Host onRender={tally.onRender} sample="fleet" />,
            async (visitor) => {
                await visitor.see(button(MAC));
                await settled();
                tally.taken();

                // When - a Mac is opened, its exits left to end, then one of its machines
                await visitor.click(button(MAC));
                await settled();
                await visitor.gone(button(LEFT_BEHIND));
                taken.push(tally.taken());
                await visitor.click(button(MACHINE));
                await settled();
                taken.push(tally.taken());
            },
        );

        // Then - each costs its pinned count
        expect(taken).toStrictEqual([
            {
                by: { Canvas: 2, CanvasNodeOf: 2, Host: 1, Icon: 1, Marks: 1, You: 3 },
                commits: 2,
                renders: 10,
            },
            {
                by: { Canvas: 1, CanvasNodeOf: 7, Host: 1, Marks: 10, You: 5 },
                commits: 1,
                renders: 24,
            },
        ]);
    });

    test('a pan and a wheel render no component', async () => {
        // Given - the fleet at its root, its renders counted
        const tally = tallied();
        let panned: Tally | undefined;
        await component.render(
            <Host onRender={tally.onRender} sample="fleet" />,
            async (visitor) => {
                await visitor.see(button(MAC));
                await settled();
                tally.taken();

                // When - the desk is dragged, then wheeled
                await visitor.drag(button(MAC), { x: 0.5, y: 0.5 }, { x: 0.2, y: 3 });
                document.querySelector('[role="group"]')?.dispatchEvent(
                    new WheelEvent('wheel', {
                        bubbles: true,
                        cancelable: true,
                        deltaX: 30,
                        deltaY: 50,
                    }),
                );
                await settled();
                panned = tally.taken();
            },
        );

        // Then - nothing rendered
        expect(panned).toStrictEqual({ by: {}, commits: 0, renders: 0 });
    });
});
