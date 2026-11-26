import {
    banner,
    button,
    content,
    dialog,
    disabled,
    enabled,
    field,
    heading,
    main,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const RESTARTED = 'Sun 18 Oct. Down from 04:30 to 04:34, out of memory; restarted 1 time';

describe('server', () => {
    test('S1 a healthy server reads its four steps and its latency', async () => {
        // Given - the window on infra's page over two-macs, a server whose check passes
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Overview',
            async (visitor) => {
                // When - Health is read
                const health = region('Health');
                await visitor.see(health);

                // Then - its address and SSH each read their answer
                await visitor.see(within(health, content('192.168.64.5')));
                await visitor.see(within(health, content('answers')));
                await visitor.see(
                    within(region('Check latency'), content('41 ms', { exact: false })),
                );
            },
        );

        // And - its verdict, its state and its check are calm, each a mark named by its word
        expect(result.html.text).toContain('aria-label="Healthy."');
        expect(result.html.text).toContain('aria-label="healthy"');
        expect(result.html.text).toContain('aria-label="passes"');
        expect(result.errors.text).toBe('');
    });

    test("S1 a server's last 30 days are the engine's strip, its sentence and a day named by its own", async () => {
        // Given - the window on infra's page over working-hard, a server restarted on two of its 30 days
        const result = await website.visit(
            '/?world=working-hard&machine=infra&tab=Overview',
            async (visitor) => {
                // When - its last 30 days are read
                const days = region('Last 30 days');
                await visitor.see(days);

                // Then - the panel says the engine's sentence of the 30 days, and a restarted day its own
                await visitor.see(
                    within(days, content('Up 99.98 % over 30 days, 2 restarts', { exact: false })),
                );
                await visitor.see(within(days, button(RESTARTED)));
            },
        );
        expect(result.errors.text).toBe('');
    });

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

    test('S3 a server down restarted from its line in Needs you leaves it, and no longer reads down', async () => {
        // Given - mac-mini's page over stories, whose server infra fails its check
        const result = await website.visit(
            '/?world=stories&machine=mac-mini&tab=Overview',
            async (visitor) => {
                const needs = region('Needs you, on this Mac');
                const line = content('infra check failing since 10:18', { exact: false });
                await visitor.see(within(needs, line));
                await visitor.see(content('down · up 12 d'));

                // When - its Restart is pressed, which asks nothing since the server is down
                await visitor.click(within(needs, button('Restart infra', { exact: false })));
                await visitor.gone(dialog('Restart infra?'));

                // Then - the line has left Needs you for Done here, and infra no longer reads down among the mini's machines
                await visitor.gone(within(needs, line));
                await visitor.see(within(region('Done here'), line));
                await visitor.gone(content('down · up 12 d'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('S3 the restart of a server down is recorded in the History, by me', async () => {
        // Given - mac-mini's page over stories, whose server infra fails its check
        const result = await website.visit(
            '/?world=stories&machine=mac-mini&tab=Overview',
            async (visitor) => {
                const needs = region('Needs you, on this Mac');
                const line = content('infra check failing since 10:18', { exact: false });

                // When - its Restart is pressed, which asks nothing, then the Inbox opened
                await visitor.click(within(needs, button('Restart infra', { exact: false })));
                await visitor.gone(dialog('Restart infra?'));
                await visitor.gone(within(needs, line));
                await visitor.click(within(banner(), button('Inbox', { exact: false })));

                // Then - Today holds the engine's line of this restart, beside the one of the morning
                const today = region('Today');
                await visitor.see(within(today, button('10:30 infra restarted by you')));
                await visitor.see(within(today, button('09:05 infra restarted by you')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('S4 Delete names the disk it takes, asks for the name typed, and stays off until it matches', async () => {
        // Given - the window on infra's page over two-macs, a server that runs on an 80 GB disk
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Overview',
            async (visitor) => {
                // When - Delete is pressed
                await visitor.click(button('Delete infra…'));

                // Then - the title names the disk, the dialog asks for the name, and its Delete is off
                const confirm = dialog('Delete infra and its 80 GB disk?');
                await visitor.see(confirm);
                const remove = within(confirm, button('Delete'));
                await visitor.see(disabled(remove));

                // When - the name is typed
                await visitor.fill(within(confirm, field('Type infra to confirm')), 'infra');

                // Then - Delete is on
                await visitor.see(enabled(remove));
            },
        );
        expect(result.html.text).toMatch(/class="[^"]*destroy[^"]*"[^>]*>Delete</u);
        expect(result.errors.text).toBe('');
    });

    test('S4 Restart of a server that runs asks yes or no, no name, and Cancel leaves it', async () => {
        // Given - the window on infra's page over two-macs, a server that runs
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Overview',
            async (visitor) => {
                // When - Restart is pressed
                await visitor.click(button('Restart'));

                // Then - the dialog asks yes or no: its Restart is on, and no name is asked
                const confirm = dialog('Restart infra?');
                await visitor.see(confirm);
                const restart = within(confirm, button('Restart'));
                await visitor.see(enabled(restart));
                await visitor.gone(within(confirm, field('Type infra to confirm')));

                // When - Cancel is pressed
                await visitor.click(within(confirm, button('Cancel')));

                // Then - the dialog is gone and infra still runs
                await visitor.gone(confirm);
                await visitor.see(heading('infra', { exact: false }));
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
                const remove = dialog('Delete infra and its 80 GB disk?');
                await visitor.see(
                    within(
                        remove,
                        content('infra and everything on its disk are deleted for good.'),
                    ),
                );
                await visitor.gone(
                    within(
                        remove,
                        content('Always on. Stopping it asks yes or no.', { exact: false }),
                    ),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('S5 a restarted day of the strip opens the Inbox on its history', async () => {
        // Given - the window on infra's page over working-hard
        const result = await website.visit(
            '/?world=working-hard&machine=infra&tab=Overview',
            async (visitor) => {
                const restarted = within(region('Last 30 days'), button(RESTARTED));
                await visitor.see(restarted);

                // When - the restarted day is pressed
                await visitor.click(restarted);

                // Then - the Inbox opens on its History
                await visitor.see(region('History'));
            },
        );
        expect(result.errors.text).toBe('');
    });

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
