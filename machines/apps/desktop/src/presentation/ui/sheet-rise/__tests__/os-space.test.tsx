import { button, component, file, navigation, region, waitUntil } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { tallied } from '../../../screens/__tests__/tally';
import type { Tally } from '../../../screens/__tests__/tally';
import { reducedMotion, settled } from '../../canvas/__tests__/canvas-host';
import {
    BAR,
    DEEP,
    expanded,
    focusedNode,
    isRaised,
    OPEN,
    scroller,
    Space,
    strip,
    wheel,
} from './space-host';

describe('os-space', () => {
    test.each([
        { golden: './__expected__/os-space-lowered.aria.yaml', name: 'lowered', raised: false },
        { golden: './__expected__/os-space-raised.aria.yaml', name: 'raised', raised: true },
    ] as const)('$name: the tree is the golden', async ({ golden, raised }) => {
        // Given - the space over a section of Boundaries, under reduced motion
        const restore = reducedMotion(true);
        const result = await component.render(
            <Space raised={raised} start={DEEP} />,
            async (visitor) => {
                // When - it has drawn
                await visitor.see(raised ? navigation(BAR) : button(OPEN));
                await settled();
            },
        );
        restore();

        // Then - lowered, the canvas and the strip named Open; raised, the bar and the page's region alone
        await expect(result.tree).toMatch(file(golden));
        expect(result.errors.text).toBe('');
    });

    test('a click on the strip raises the page, moves the focus into it, and Escape hands it back to the node', async () => {
        // Given - the space with a page open, under reduced motion
        const restore = reducedMotion(true);
        const seen: {
            raised: boolean;
            inert: boolean;
            scroll: boolean;
            node?: string | undefined;
        }[] = [];
        await component.render(<Space />, async (visitor) => {
            await visitor.see(button(OPEN));

            // When - the strip is clicked, then Escape pressed
            await visitor.click(button(OPEN));
            await visitor.see(region('Boundaries'));
            await waitUntil(() => document.activeElement === scroller(), {
                why: 'the page took the focus',
            });
            seen.push({
                inert:
                    document.querySelector('[data-node]')?.closest<HTMLElement>('[inert]') !== null,
                raised: isRaised(),
                scroll: document.activeElement === scroller(),
            });
            await visitor.press('Escape');
            await waitUntil(() => focusedNode() !== undefined, {
                why: 'the canvas took the focus back',
            });
            seen.push({ inert: false, node: focusedNode(), raised: isRaised(), scroll: false });
        });
        restore();

        // Then - raised, the canvas is inert and the page holds the focus; lowered, the page's node holds it
        expect(seen).toStrictEqual([
            { inert: true, raised: true, scroll: true },
            { inert: false, node: 'P:product/boundaries', raised: false, scroll: false },
        ]);
    });

    test('escape lowers the page first, keeping the section read open, and only then climbs the canvas', async () => {
        // Given - a section read, raised, the page holding the focus
        const restore = reducedMotion(true);
        const seen: { raised: boolean; section: boolean; page: boolean }[] = [];
        const look = () => {
            seen.push({
                page: expanded('P:product/boundaries'),
                raised: isRaised(),
                section: expanded('C:product/boundaries:2'),
            });
        };
        await component.render(<Space raised start={DEEP} />, async (visitor) => {
            await visitor.see(navigation(BAR));
            scroller().focus();

            // When - Escape, then Escape again
            await visitor.press('Escape');
            await waitUntil(() => focusedNode() !== undefined, {
                why: 'the canvas took the focus back',
            });
            look();
            await visitor.press('Escape');
            await waitUntil(() => !expanded('C:product/boundaries:2'), {
                why: 'the canvas went up',
            });
            look();
        });
        restore();

        // Then - the first lowers onto the section, the second closes the section on the canvas
        expect(seen).toStrictEqual([
            { page: true, raised: false, section: true },
            { page: true, raised: false, section: false },
        ]);
    });

    test('the bar: a sibling page reads that page, Canvas lowers, a seat pill lowers onto the seat', async () => {
        // Given - a section of Boundaries read
        const restore = reducedMotion(true);
        const seen: { raised: boolean; label: null | string }[] = [];
        const look = () => {
            seen.push({
                label: document.querySelector('section')?.getAttribute('aria-label') ?? null,
                raised: isRaised(),
            });
        };
        await component.render(<Space raised start={DEEP} />, async (visitor) => {
            await visitor.see(navigation(BAR));

            // When - a sibling page's dot, then Canvas, then raised again and the seat's pill
            await visitor.click(button('The stack, 1 waits for you'));
            await settled();
            look();
            await visitor.click(button('Canvas'));
            await settled();
            look();
            await visitor.click(button('Open The stack'));
            await visitor.see(navigation(BAR));
            await visitor.click(button('Produit, 4 wait for you'));
            await settled();
            look();
        });
        restore();

        // Then - the dot kept it raised on the new page, Canvas and the pill lowered it, the pill onto the seat
        expect(seen).toStrictEqual([
            { label: 'The stack', raised: true },
            { label: 'The stack', raised: false },
            { label: 'Boundaries', raised: false },
        ]);
    });

    test('raising and lowering render no canvas component, a wheel that raises only the state holders', async () => {
        // Given - the space with a page open, drawn to rest, its renders counted
        const restore = reducedMotion(true);
        const tally = tallied();
        const taken: Tally[] = [];
        await component.render(<Space onRender={tally.onRender} />, async (visitor) => {
            await visitor.see(button(OPEN));
            await settled();
            await settled();
            await settled();
            tally.taken();

            // When - wheeled up, lowered by Escape, then raised by a click
            wheel(strip(), 40);
            await settled();
            taken.push(tally.taken());
            await visitor.press('Escape');
            await waitUntil(() => focusedNode() !== undefined, {
                why: 'the canvas took the focus back',
            });
            taken.push(tally.taken());
            await visitor.click(button(OPEN));
            await settled();
            taken.push(tally.taken());
        });
        restore();

        // Then - each costs its pinned count, the canvas never among them
        expect(taken).toStrictEqual([
            { by: { OsSpaceScene: 1, Sheet: 1 }, commits: 1, renders: 2 },
            { by: { OsSpaceScene: 1, Sheet: 1 }, commits: 1, renders: 2 },
            { by: { OsSpaceScene: 1, Sheet: 1 }, commits: 1, renders: 2 },
        ]);
    });
});
