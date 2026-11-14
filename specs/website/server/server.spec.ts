import { button, content, dialog, disabled, enabled, field, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('server', () => {
    test('S1 a healthy server reads its four steps and its latency', async () => {
        // Given - the window on infra's page over two-macs, a server whose check passes
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Overview',
            async (visitor) => {
                // When - Health is read
                const health = region('Health');
                await visitor.see(health);

                // Then - its state, its address, SSH and its check each read their answer
                await visitor.see(within(health, content('Running')));
                await visitor.see(within(health, content('192.168.64.5')));
                await visitor.see(within(health, content('Answers')));
                await visitor.see(within(health, content('Passes')));
                await visitor.see(
                    within(region('Check latency'), content('41 ms', { exact: false })),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "S1 waits for a verdict over a server's state, address, SSH and check on machines.list, and for 30 days of uptime on MachinesUptimeDay",
    );

    test.todo(
        "S2 waits for a verdict over a server's state, address, SSH and check on machines.list",
    );

    test.todo(
        'S3 waits for a recording of a Restart of a server that is down, and of the line it leaves',
    );

    test('S4 Delete asks for the name typed, and stays off until it matches', async () => {
        // Given - the window on infra's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Overview',
            async (visitor) => {
                // When - Delete is pressed
                await visitor.click(button('Delete infra…'));

                // Then - the dialog asks for the name, and its Delete is off
                const confirm = dialog('Delete infra?');
                await visitor.see(confirm);
                const remove = within(confirm, button('Delete'));
                await visitor.see(disabled(remove));

                // When - the name is typed
                await visitor.fill(within(confirm, field('Type infra to confirm')), 'infra');

                // Then - Delete is on
                await visitor.see(enabled(remove));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'S4 window differs: the Restart dialog of a server asks no name typed and says nothing of what stops, where the row asks both',
    );

    test.todo('S5 waits for a state per day on MachinesUptimeDay');

    test.todo('S6 waits for the accounts redesign');
});
