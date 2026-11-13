import { button, component, content } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test } from 'vitest';

import { LineList } from '../line-list';

const LINES = Array.from(
    { length: 5000 },
    (_, index) => `console line ${String(index + 1)} of the boot`,
);

function drawn(html: string): number {
    return html.match(/console line \d+ of the boot/gu)?.length ?? 0;
}

function Tabs() {
    const [here, setHere] = useState(true);
    return (
        <>
            <button
                onClick={() => {
                    setHere((was) => !was);
                }}
                type="button"
            >
                {here ? 'Leave' : 'Return'}
            </button>
            {here ? <LineList lines={LINES} memory="kept" query="" /> : <p>Another tab</p>}
        </>
    );
}

describe('LineList', () => {
    test('5 000 lines keep a window of them in the page, and End reaches the last', async () => {
        // Given - a console of 5 000 lines
        const result = await component.render(
            <LineList lines={LINES} memory="deep" query="" />,
            async (visitor) => {
                await visitor.see(content('console line 1 of the boot'));

                // When - the list takes the keyboard and End is pressed
                await visitor.click(content('console line 1 of the boot'));
                await visitor.press('End');

                // Then - the last line is drawn, the first is not
                await visitor.see(content('console line 5000 of the boot'));
                await visitor.gone(content('console line 1 of the boot'));
            },
        );

        // Then - the page holds a window of the lines, never all of them
        expect(drawn(result.html.text)).toBeGreaterThan(20);
        expect(drawn(result.html.text)).toBeLessThan(80);
        expect(result.content.text).toContain('5000 lines');
    });

    test('a filter keeps the box as tall as before and counts what it left', async () => {
        // Given - the same console filtered to the lines holding 4999
        const result = await component.render(
            <LineList lines={LINES} memory="filtered" query="line 4999" />,
            async (visitor) => {
                await visitor.see(content('console line 4999 of the boot'));
            },
        );

        // Then - one line of 5 000 is shown, in a box of the height the whole log takes
        expect(drawn(result.html.text)).toBe(1);
        expect(result.content.text).toContain('1 of 5000 lines');
        expect(result.html.text).toContain('height: 546px');
    });

    test('a list left and returned to is where it was left', async () => {
        // Given - the console scrolled to its end
        const result = await component.render(<Tabs />, async (visitor) => {
            await visitor.click(content('console line 1 of the boot'));
            await visitor.press('End');
            await visitor.see(content('console line 5000 of the boot'));

            // When - its tab is left, then returned to
            await visitor.click(button('Leave'));
            await visitor.see(content('Another tab'));
            await visitor.click(button('Return'));

            // Then - the end of the console is shown again, not its start
            await visitor.see(content('console line 5000 of the boot'));
        });
        expect(result.html.text).not.toContain('console line 1 of the boot<');
    });
});
