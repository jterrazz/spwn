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
