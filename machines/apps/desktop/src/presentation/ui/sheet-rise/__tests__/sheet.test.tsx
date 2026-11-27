import { button, component, navigation, waitUntil } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { reducedMotion, settled } from '../../canvas/__tests__/canvas-host';
import { BAR, elapsed, isRaised, OPEN, pointer, scroller, Space, strip, wheel } from './space-host';

describe('sheet', () => {
    test('a wheel over the strip raises past 30 px within one gesture, and never scrolls what holds the space', async () => {
        // Given - the space lowered
        const restore = reducedMotion(true);
        const seen: { raised: boolean; held: boolean }[] = [];
        await component.render(<Space />, async (visitor) => {
            await visitor.see(button(OPEN));

            // When - 20 px, a 300 ms pause, 20 px, then 15 px at once
            for (const [delta, pause] of [
                [20, 300],
                [20, 0],
                [15, 0],
            ] as const) {
                const event = wheel(strip(), delta);
                await elapsed(pause);
                await settled();
                seen.push({ held: event.defaultPrevented, raised: isRaised() });
            }
        });
        restore();

        // Then - only the sum of one gesture past 30 raises, and each wheel is kept from the page
        expect(seen).toStrictEqual([
            { held: true, raised: false },
            { held: true, raised: false },
            { held: true, raised: true },
        ]);
    });

    test('a wheel down past the canvas bottom raises the page', async () => {
        // Given - the space lowered on a page
        const restore = reducedMotion(true);
        let raised = false;
        await component.render(<Space />, async (visitor) => {
            await visitor.see(button(OPEN));

            // When - the canvas is wheeled down, far past its bottom, in one gesture
            for (const _ of Array.from({ length: 12 }, () => 0)) {
                wheel(document.querySelector('[role="group"]'), 200);
                await settled();
            }
            raised = isRaised();
        });
        restore();

        // Then - the sheet rose
        expect(raised).toBe(true);
    });

    test('a pull at the page top lowers it; the same pull after a scroll does not', async () => {
        // Given - the page raised, long enough to scroll
        const restore = reducedMotion(true);
        const seen: boolean[] = [];
        await component.render(<Space raised />, async (visitor) => {
            await visitor.see(navigation(BAR));
            const page = scroller();

            // When - scrolled, it is pulled 60 px twice; then, back at the top after a pause, again
            page.scrollTop = 200;
            wheel(page, -60);
            page.scrollTop = 0;
            wheel(page, -60);
            wheel(page, -60);
            await settled();
            seen.push(isRaised());
            await elapsed(300);
            wheel(page, -60);
            wheel(page, -60);
            await settled();
            seen.push(isRaised());
        });
        restore();

        // Then - only the pull that stayed at the top lowered it
        expect(seen).toStrictEqual([true, false]);
    });

    test('a drag of the strip raises past its distance, a short slow one settles back', async () => {
        // Given - the space lowered
        const restore = reducedMotion(true);
        const seen: { raised: boolean; transform: string }[] = [];
        await component.render(<Space />, async (visitor) => {
            await visitor.see(button(OPEN));
            const held = strip();
            const sheet = held.closest('section');

            // When - dragged up 30 px over 300 ms and released, then dragged up 80 px
            pointer('pointerdown', held, 800);
            await elapsed(150);
            pointer('pointermove', held, 785);
            await elapsed(150);
            pointer('pointermove', held, 770);
            pointer('pointerup', held, 770);
            await settled();
            seen.push({ raised: isRaised(), transform: sheet?.style.transform ?? '' });
            await visitor.drag(button(OPEN), { x: 0.5, y: 0.5 }, { x: 0.5, y: -0.25 });
            await settled();
            seen.push({ raised: isRaised(), transform: sheet?.style.transform ?? '' });
        });
        restore();

        // Then - the short one rests lowered, the long one raised, neither left an inline transform
        expect(seen).toStrictEqual([
            { raised: false, transform: '' },
            { raised: true, transform: '' },
        ]);
    });

    test('a drag of the grabber down lowers the raised page, and the click it ends in is swallowed', async () => {
        // Given - the page raised
        const restore = reducedMotion(true);
        let raised = true;
        await component.render(<Space raised />, async (visitor) => {
            await visitor.see(navigation(BAR));

            // When - its grabber is dragged 100 px down
            await visitor.drag(
                button('Raise or lower the page'),
                { x: 0.5, y: 0.5 },
                { x: 0.5, y: 5 },
            );
            await settled();
            await waitUntil(() => !isRaised(), { why: 'the page lowered' });
            await settled();
            raised = isRaised();
        });
        restore();

        // Then - it lowered and stayed lowered
        expect(raised).toBe(false);
    });

    test("a click on the bar's ground lowers the raised page; a click on a dot in it reads and keeps it raised", async () => {
        // Given - the page raised
        const restore = reducedMotion(true);
        const seen: boolean[] = [];
        await component.render(<Space raised />, async (visitor) => {
            await visitor.see(navigation(BAR));

            // When - a sibling page's dot is clicked, then the bar's own ground
            await visitor.click(button('The stack, 1 waits for you'));
            await settled();
            seen.push(isRaised());
            document.querySelector<HTMLElement>(`nav[aria-label="${BAR}"]`)?.click();
            await settled();
            seen.push(isRaised());
        });
        restore();

        // Then - the dot kept it raised, the ground lowered it
        expect(seen).toStrictEqual([true, false]);
    });

    test('with motion, the rise is one animation of the sheet; under reduced motion it jumps', async () => {
        // Given - the space lowered, first with motion, then without
        const counts: number[] = [];
        for (const reduce of [false, true]) {
            const restore = reducedMotion(reduce);
            await component.render(<Space />, async (visitor) => {
                await visitor.see(button(OPEN));

                // When - the strip is clicked
                await visitor.click(button(OPEN));
                counts.push(strip().closest('section')?.getAnimations().length ?? -1);
                await settled();
            });
            restore();
        }

        // Then - one animation with motion, none without
        expect(counts).toStrictEqual([1, 0]);
    });

    test('the page scrolls inside the sheet only', async () => {
        // Given - the page raised
        const restore = reducedMotion(true);
        let contained = '';
        await component.render(<Space raised />, async (visitor) => {
            await visitor.see(navigation(BAR));
            contained = getComputedStyle(scroller()).overscrollBehaviorY;
        });
        restore();

        // Then - its scroll never chains to what holds it
        expect(contained).toBe('contain');
    });
});
