import { button, content, field, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const FEWER_THAN_TWO = 'No chart: the engine kept fewer than two samples of it over this window.';

function curves(span: string): string[] {
    return [
        `img "CPU of linux-1 over ${span}"`,
        `img "Memory of linux-1 over ${span}"`,
        `img "Disk read of linux-1, per sample over ${span}"`,
        `img "Network in of linux-1, per sample over ${span}"`,
    ];
}

function curvesOf(tree: string): string[] {
    return [...tree.matchAll(/img "[^"]+ of linux-1[^"]*"/gu)].map((found) => found[0]);
}

describe('activity', () => {
    test('W10 Busiest now lists the processes that eat its CPU, with their share', async () => {
        // Given - the window on linux-1's Overview over two-macs, where node and vite run
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Overview',
            async (visitor) => {
                // When - Busiest now is read
                const busiest = region('Busiest now');
                await visitor.see(busiest);

                // Then - each process comes with its share of the CPU
                await visitor.see(within(busiest, content('node')));
                await visitor.see(within(busiest, content('6 %')));
                await visitor.see(within(busiest, content('vite')));
                await visitor.see(within(busiest, content('2 %')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W10 Busiest now keeps the three processes that eat the most CPU, with their share', async () => {
        // Given - linux-1's Overview over working-hard, where node, rustc, cc1plus, esbuild and git run, in that order by CPU
        const result = await website.visit(
            '/?world=working-hard&machine=linux-1&tab=Overview',
            async (visitor) => {
                // When - Busiest now is read
                const busiest = region('Busiest now');
                await visitor.see(within(busiest, content('node')));

                // Then - the first three are there with their share, the other two are not
                await visitor.see(within(busiest, content('39 %')));
                await visitor.see(within(busiest, content('rustc')));
                await visitor.see(within(busiest, content('21 %')));
                await visitor.see(within(busiest, content('cc1plus')));
                await visitor.see(within(busiest, content('15 %')));
                await visitor.gone(within(busiest, content('esbuild')));
                await visitor.gone(within(busiest, content('git')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W11 its log reads numbered lines, a filter narrows them, and Refresh is offered', async () => {
        // Given - linux-1's Activity over stories, where its console printed 12 lines
        const result = await website.visit(
            '/?world=stories&machine=linux-1&tab=Activity',
            async (visitor) => {
                const logs = region('What linux-1 prints');
                await visitor.see(within(logs, content('12 lines', { exact: false })));
                await visitor.see(within(logs, button('Refresh')));

                // When - a word is typed in the filter
                await visitor.fill(within(logs, field('Filter the lines')), 'virtio');

                // Then - only the lines that hold it stay
                await visitor.see(within(logs, content('1 of 12 lines', { exact: false })));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "W11 waits for a call reading a stopped VM's log, and for what its hypervisor printed",
    );

    test.todo("W25 waits for a call reading the last run's lines");

    test.todo(
        'W26 not in this version (decisions 2026-11-12, Logs tab: a reading, not a live feed)',
    );

    test('W27 Activity reads its four measures over a window of 6 min, 1 h or 24 h', async () => {
        // Given - linux-1's Activity over two-macs, a running workspace
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Activity',
            async (visitor) => {
                // When - the page is read
                // Then - the four measures are there, each with its value now
                await visitor.see(within(region('CPU'), content('8%')));
                await visitor.see(within(region('Memory'), content('1.1GB', { exact: false })));
                await visitor.see(region('Disk'));
                await visitor.see(region('Network'));

                // When - another window is chosen
                await visitor.click(button('1 h'));

                // Then - the measures are still there
                await visitor.see(region('CPU'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W27 Activity draws its four curves over the window chosen', async () => {
        // Given - linux-1's Activity over working-hard, whose Mac's engine kept a day of its activity
        const page = '/?world=working-hard&machine=linux-1&tab=Activity';
        const sixMinutes = await website.visit(page, async (visitor) => {
            // When - the page is read
            await visitor.see(within(region('CPU'), content('81%')));
            await visitor.gone(content(FEWER_THAN_TWO));
        });
        const anHour = await website.visit(page, async (visitor) => {
            await visitor.see(region('Network'));
            await visitor.gone(content(FEWER_THAN_TWO));

            // When - another window is chosen
            await visitor.click(button('1 h'));
            await visitor.see(region('Network'));
            await visitor.gone(content(FEWER_THAN_TWO));
        });

        // Then - each measure is a curve over the window shown
        expect(sixMinutes.errors.text).toBe('');
        expect(anHour.errors.text).toBe('');
        expect(curvesOf(sixMinutes.tree.text)).toStrictEqual(curves('6 min'));
        expect(curvesOf(anHour.tree.text)).toStrictEqual(curves('1 h'));
    });

    test('W27 Busiest lists the five processes by CPU, then by memory', async () => {
        // Given - linux-1's Activity over working-hard
        const result = await website.visit(
            '/?world=working-hard&machine=linux-1&tab=Activity',
            async (visitor) => {
                const busiest = region('Busiest');

                // When - the list is read by CPU
                // Then - the five are there with their share
                await visitor.see(within(busiest, content('39 %')));
                await visitor.see(within(busiest, content('21 %')));
                await visitor.see(within(busiest, content('15 %')));
                await visitor.see(within(busiest, content('9 %')));
                await visitor.see(within(busiest, content('4 %')));

                // When - it is read by memory
                await visitor.click(within(busiest, button('By memory')));

                // Then - the five are there with what each holds
                await visitor.see(within(busiest, content('2 GB')));
                await visitor.see(within(busiest, content('1.4 GB')));
                await visitor.see(within(busiest, content('0.9 GB')));
                await visitor.see(within(busiest, content('0.3 GB')));
                await visitor.see(within(busiest, content('100 MB')));
                await visitor.gone(within(busiest, content('39 %')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "W28 waits for the engine's finding of a busy CPU, and its process, on activity.read",
    );
});
