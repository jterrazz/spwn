import {
    button,
    content,
    dialog,
    disabled,
    enabled,
    field,
    main,
    region,
    within,
} from '@jterrazz/test';
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
                await visitor.see(within(health, content('Healthy.', { exact: false })));
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

    test.todo('S1 waits for 30 days of uptime on MachinesUptimeDay');

    test('S2 a server whose check fails says so, and no step of its health is drawn ok', async () => {
        // Given - the window on infra's page over stories, a server on mac-mini whose check fails
        const result = await website.visit(
            '/?world=stories&machine=infra&tab=Overview',
            async (visitor) => {
                // When - I open it
                // Then - its health says it is down, and the engine's line of the failing check
                const health = region('Health');
                await visitor.see(within(health, content('Down.')));
                await visitor.see(within(health, content('Its check fails since 10:18.')));
            },
        );
        expect(result.errors.text).toBe('');
    });

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

    test('S4 Restart asks for the name typed, and stays off until it matches', async () => {
        // Given - the window on infra's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Overview',
            async (visitor) => {
                // When - Restart is pressed
                await visitor.click(button('Restart'));

                // Then - the dialog asks for the name, and its Restart is off
                const confirm = dialog('Restart infra?');
                await visitor.see(confirm);
                const restart = within(confirm, button('Restart'));
                await visitor.see(disabled(restart));

                // When - the name is typed
                await visitor.fill(within(confirm, field('Type infra to confirm')), 'infra');

                // Then - Restart is on
                await visitor.see(enabled(restart));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("S4 the dialog of Restart and of Delete says what stops, in the engine's sentence", async () => {
        // Given - the window on infra's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Overview',
            async (visitor) => {
                // When - Restart is pressed
                await visitor.click(button('Restart'));

                // Then - its dialog says what the restart stops, then Cancel leaves it
                const restart = dialog('Restart infra?');
                await visitor.see(
                    within(
                        restart,
                        content('It is a server: what it serves stops while it restarts.'),
                    ),
                );
                await visitor.click(within(restart, button('Cancel')));

                // When - Delete is pressed
                await visitor.click(button('Delete infra…'));

                // Then - its dialog says what the delete ends, and nothing of a hold that does not argue against it
                const remove = dialog('Delete infra?');
                await visitor.see(
                    within(
                        remove,
                        content('infra and everything on its disk are deleted for good.'),
                    ),
                );
                await visitor.gone(
                    within(
                        remove,
                        content('Always on. Stopping it needs its name typed.', { exact: false }),
                    ),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('S5 waits for a state per day on MachinesUptimeDay');

    test('S6 a server has no Sign-ins tab: it keeps its own sign-ins', async () => {
        // Given - the window on infra's page over two-macs, a server
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Overview',
            async (visitor) => {
                // When - its tabs are read
                // Then - Overview is there and Sign-ins is not
                await visitor.see(within(main(), content('Overview')));
                await visitor.gone(within(main(), content('Sign-ins')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
