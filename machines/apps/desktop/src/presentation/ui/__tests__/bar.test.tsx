import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Bar, ShareBar } from '../bar';

describe('Bar', () => {
    test.each([
        ['an unknown share', Number.NaN],
        ['a share of nothing', Number.POSITIVE_INFINITY],
    ])('%s draws an empty bar', async (_, percent) => {
        // Given - a bar handed a percent no total gave
        const result = await component.render(<Bar label="Memory" percent={percent} />);

        // Then - the bar is empty and reads zero, never NaN
        expect(result.html.text).toContain('--fill: 0%');
        expect(result.html.text).toContain('aria-valuenow="0"');
        expect(result.html.text).not.toMatch(/NaN|Infinity/u);
    });

    test('a share bar draws a part by its step, and its free well in orange only when it is low', async () => {
        // Given - a whole of two kinds and a free well, once at rest and once low
        const parts = [
            { key: 'apps', size: 10, step: 't2' as const },
            { key: 'data', size: 20, step: 's3' as const },
            { key: 'free', rest: true, size: 70 },
        ];
        const calm = await component.render(<ShareBar label="Disk" of={100} parts={parts} />);
        const low = await component.render(<ShareBar label="Disk" low of={100} parts={parts} />);

        // Then - each kind carries its step, and only the low bar carries the low look
        expect(calm.html.text).toContain('data-step="t2"');
        expect(calm.html.text).toContain('data-step="s3"');
        expect(calm.html.text).not.toMatch(/class="[^"]*low/u);
        expect(low.html.text).toMatch(/class="[^"]*low/u);
    });

    test('a share bar of a total of zero draws its parts empty', async () => {
        // Given - parts measured against a total of zero
        const result = await component.render(
            <ShareBar label="Disk" of={0} parts={[{ key: 'apps', size: 3 }]} />,
        );

        // Then - each part takes no room, never NaN
        expect(result.html.text).toContain('--share: 0.00%');
        expect(result.html.text).not.toMatch(/NaN|Infinity/u);
    });
});
