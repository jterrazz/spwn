import { button, content, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const FEWER_THAN_TWO = 'No chart: the engine kept fewer than two samples of it over this window.';

describe('activity', () => {
    test('M11 Activity reads its four measures, its machines among the busiest, its heat and battery', async () => {
        // Given - the Activity tab of This Mac over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Activity',
            async (visitor) => {
                // When - the page is read
                // Then - the four measures, the busiest and the Mac's own facts are there
                await visitor.see(region('CPU'));
                await visitor.see(region('Memory'));
                await visitor.see(region('Disk'));
                await visitor.see(region('Network'));
                await visitor.see(within(region('This Mac'), content('96 % · charging')));
                await visitor.see(within(region('This Mac'), content('Nominal')));

                // When - its machine among the busiest is pressed
                await visitor.click(
                    within(
                        region('Busiest'),
                        button('linux-1 a machine of this Mac', { exact: false }),
                    ),
                );

                // Then - that machine opens
                await visitor.see(heading('linux-1 running'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M11 Activity draws its four curves from the history its engine kept', async () => {
        // Given - the Activity tab of This Mac over stories, whose engine kept a day of its activity
        const result = await website.visit(
            '/?world=stories&machine=macbook-pro&tab=Activity',
            async (visitor) => {
                // When - the page is read
                await visitor.see(region('Network'));
                await visitor.gone(content(FEWER_THAN_TWO));
            },
        );

        // Then - each measure is a curve over the last 6 min
        expect(result.errors.text).toBe('');
        expect(result.tree.text).toContain('img "CPU of macbook-pro over 6 min"');
        expect(result.tree.text).toContain('img "Memory of macbook-pro over 6 min"');
        expect(result.tree.text).toContain('img "Disk read of macbook-pro, per sample over 6 min"');
        expect(result.tree.text).toContain(
            'img "Network in of macbook-pro, per sample over 6 min"',
        );
    });
});
