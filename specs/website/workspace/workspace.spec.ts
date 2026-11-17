import {
    button,
    content,
    dialog,
    disabled,
    field,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');

describe('workspace', () => {
    test.todo(
        'W1 waits for the Tauri window, the only one that opens Terminal, and for the ssh command to be copied beside Open in Terminal',
    );

    test('W2 a workspace says what it is for, who created it and when', async () => {
        // Given - the window on linux-5 over creating, made by a session
        const result = await website.visit(
            '/?world=creating&machine=linux-5&tab=Overview',
            async (visitor) => {
                // When - Why it exists is read
                const why = region('Why it exists');
                await visitor.see(why);

                // Then - its purpose, its creator and its date are there
                await visitor.see(within(why, content('Lot shell, one block per package')));
                await visitor.see(within(why, content('4 Nov, by studio-os-fe')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W2 a workspace says who holds it and whether it is safe to delete', async () => {
        // Given - the window on the stories world, where linux-1 is held by a session and review-bot, persistent, by none
        const held = await website.visit(
            '/?world=stories&machine=linux-1&tab=Overview',
            async (visitor) => {
                // When - Why it exists is read
                // Then - its hold names the session and says it is not safe to delete
                await visitor.see(
                    within(
                        region('Why it exists'),
                        content('It is held by its lot, by studio-os-fe: work may be in progress.'),
                    ),
                );
            },
        );
        const yours = await website.visit(
            '/?world=stories&machine=review-bot&tab=Overview',
            async (visitor) => {
                // Then - the hold of one no session holds says it is yours, and stays until you delete it
                await visitor.see(
                    within(region('Why it exists'), content('Yours. Stays until you delete it.')),
                );
            },
        );
        expect(held.errors.text).toBe('');
        expect(yours.errors.text).toBe('');
    });

    test('W3 Delete of a held workspace says who holds it before it asks for its name', async () => {
        // Given - the window on linux-1 over stories, where linux-1 is held by its lot, by studio-os-fe
        const result = await website.visit(
            '/?world=stories&machine=linux-1&tab=Overview',
            async (visitor) => {
                // When - Delete is pressed
                await visitor.click(button('Delete linux-1…'));

                // Then - the dialog says who holds it and asks for the name, its Delete off
                const confirm = dialog('Delete linux-1?');
                await visitor.see(
                    within(
                        confirm,
                        content(
                            'It is held by its lot, by studio-os-fe: work may be in progress.',
                            {
                                exact: false,
                            },
                        ),
                    ),
                );
                await visitor.see(within(confirm, field('Type linux-1 to confirm')));
                const remove = within(confirm, button('Delete'));
                await visitor.see(disabled(remove));
                await visitor.see(within(confirm, button('Cancel')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W4 Stop reads Stopped everywhere, and the memory it held is free on Home', async () => {
        // Given - the window on Home over two-macs, where linux-1 runs and holds 8 GB of its Mac's 16
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            await visitor.see(within(region('Machines'), content('4 free')));
            await visitor.click(within(sidebar, button('linux-1 Running')));
            await visitor.see(heading('linux-1 Running'));

            // When - Stop is pressed and confirmed
            await visitor.click(button('Stop'));
            await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));

            // Then - it reads Stopped, and Home counts its memory free
            await visitor.see(heading('linux-1 Stopped'));
            await visitor.see(within(sidebar, button('linux-1 Stopped')));
            await visitor.click(within(sidebar, button('Home')));
            await visitor.see(within(region('Machines'), content('12 free')));
        });
        expect(result.errors.text).toBe('');
    });

    test('W4 Start brings a stopped workspace back', async () => {
        // Given - the window on api over two-macs, a workspace that is stopped
        const result = await website.visit('/?world=two-macs&machine=api', async (visitor) => {
            await visitor.see(heading('api Stopped'));

            // When - Start is pressed
            await visitor.click(button('Start'));

            // Then - it reads Running
            await visitor.see(heading('api Running'));
            await visitor.see(within(sidebar, button('api Running')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'W4 waits for a recording of activity.read after a Stop, and of a Start after it on the same machine',
    );

    test('W5 a VM being created reads Creating, with its step', async () => {
        // Given - the window on linux-4 over creating, which is booting
        const result = await website.visit(
            '/?world=creating&machine=linux-4&tab=Overview',
            async (visitor) => {
                // When - the page is read
                // Then - it reads Creating and names the step it is at
                await visitor.see(heading('linux-4 Creating'));
                await visitor.see(content('Creating: boot 2 of 4'));
                await visitor.see(disabled(button('Open in Terminal')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('W5 waits for a recording of a failed creation, its step and its log');

    test('W6 a password granted to a VM is listed among its grants', async () => {
        // Given - GitHub deploy's page over two-macs, granted nowhere, linux-1 signed out of gh
        const result = await website.visit(
            '/?world=two-macs&page=passwords&item=pw-1&pace=600',
            async (visitor) => {
                await visitor.click(button('Grant…'));
                const sheet = dialog('Grant GitHub deploy');

                // When - linux-1 is picked and granted
                await visitor.click(within(sheet, button('linux-1', { exact: false })));
                await visitor.click(within(sheet, button('Grant to linux-1')));

                // Then - the sheet goes, and linux-1 holds it
                await visitor.gone(sheet);
                await visitor.see(within(region('Granted to'), content('linux-1')));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('until');
    });

    test('W30 a pasted code goes through a field the page never holds, and a refused one is said', async () => {
        // Given - linux-1's Sign-ins over sign-ins, claude signed out and its login asking a code back
        const result = await website.visit(
            '/?world=sign-ins&machine=linux-1&tab=Sign-ins&pace=600',
            async (visitor) => {
                await visitor.click(within(region('Sign-ins'), button('Sign in claude')));
                const sheet = dialog('Sign in to claude on linux-1');
                await visitor.see(
                    within(sheet, content('then paste the code it gives you', { exact: false })),
                );

                // When - a code the provider refuses is sent
                await visitor.fill(within(sheet, field('Code from claude.com')), 'nope');
                await visitor.click(within(sheet, button('Continue')));

                // Then - the engine's refusal in its words, and the field is empty for another
                await visitor.see(within(sheet, content('the code was refused', { exact: false })));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('nope');
    });

    test('W31 an expired code is said, and New code starts the sign-in again', async () => {
        // Given - linux-1's Sign-ins over sign-ins, claude's sign-in opened and its code never pasted
        const result = await website.visit(
            '/?world=sign-ins&machine=linux-1&tab=Sign-ins&pace=600',
            async (visitor) => {
                await visitor.click(within(region('Sign-ins'), button('Sign in claude')));
                const sheet = dialog('Sign in to claude on linux-1');

                // When - the engine lets the code run out
                await visitor.see(
                    within(sheet, content('expired before Claude was approved', { exact: false })),
                );

                // Then - New code starts again, and the site is offered
                await visitor.click(within(sheet, button('New code')));
                await visitor.see(within(sheet, button('Open claude.com')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("W32 a far Mac's VM says to sign in from its Mac and offers no Sign in", async () => {
        // Given - linux-3's Sign-ins over stories, a VM of the Mac mini
        const result = await website.visit(
            '/?world=stories&machine=linux-3&tab=Sign-ins',
            async (visitor) => {
                // When - its rows are read
                const rows = region('Sign-ins');
                await visitor.see(within(rows, heading('Sign-ins 4')));

                // Then - no row offers Sign in, whichever the program
                await visitor.gone(within(rows, button('Sign in gh')));
                await visitor.gone(within(rows, button('Sign in claude')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
