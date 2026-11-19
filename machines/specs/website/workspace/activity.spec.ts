import { button, content, field, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

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

    test.todo('W10 waits for a recording of a VM with three processes running');

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

    test.todo('W26 waits for a machines.log watch that streams each new line');

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

    test.todo(
        'W27 waits for a recording of enough samples for the four curves, and of five busiest processes by CPU and by memory',
    );

    test.todo(
        "W28 waits for the engine's finding of a busy CPU, and its process, on activity.read",
    );
});
