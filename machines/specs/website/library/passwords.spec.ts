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

const SHEET = dialog('Add a password');

describe('passwords', () => {
    test("D27 the sidebar's Passwords lists this Mac's vault under the engine's verdict", async () => {
        // Given - the window on Home over two-macs
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            // When - Passwords is pressed in the sidebar's Library
            await visitor.click(within(navigation('Sidebar'), button('Passwords')));

            // Then - the page names this Mac, and each password says what it is and where it is granted
            await visitor.see(heading('Passwords on macbook-pro'));
            await visitor.see(content('3 kept, 1 granted', { exact: false }));
            const passwords = region('Passwords');
            await visitor.see(within(passwords, content('fine-grained token')));
            await visitor.see(within(passwords, content('1 machine')));
        });
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('Copy');
    });

    test('D28 a password opens on its facts, its value withheld, and its grants', async () => {
        // Given - the Passwords page over two-macs
        const result = await website.visit('/?world=two-macs&page=passwords', async (visitor) => {
            // When - npm publish is opened
            await visitor.click(
                within(region('Passwords'), button('npm publish', { exact: false })),
            );

            // Then - its value is withheld, and its grant to linux-1 carries a Remove
            await visitor.see(heading('npm publish'));
            await visitor.see(within(region('This password'), content('Withheld')));
            const grants = region('Granted to');
            await visitor.see(within(grants, button('Remove npm publish from linux-1')));
        });
        expect(result.errors.text).toBe('');
    });

    test('D29 removing a password counts the grants that end, and it leaves the list', async () => {
        // Given - npm publish's page over two-macs, granted to linux-1
        const result = await website.visit(
            '/?world=two-macs&page=passwords&item=pw-2',
            async (visitor) => {
                // When - Remove password is pressed and confirmed
                await visitor.click(button('Remove password'));
                const asked = dialog('Remove npm publish?');
                await visitor.see(within(asked, content('Its grants end: 1 machine.')));
                await visitor.click(within(asked, button('Remove')));

                // Then - the vault is listed again without it
                await visitor.see(heading('Passwords on macbook-pro'));
                await visitor.gone(
                    within(region('Passwords'), button('npm publish', { exact: false })),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D31 a grant refused at the presence check is said in the sheet, which stays open', async () => {
        // Given - GitHub deploy's page over presence-refused, whose Mac does not confirm the owner is there
        const result = await website.visit(
            '/?world=presence-refused&page=passwords&item=pw-1&pace=600',
            async (visitor) => {
                await visitor.click(button('Grant…'));
                const sheet = dialog('Grant GitHub deploy');
                await visitor.click(within(sheet, field('linux-1')));

                // When - the grant to linux-1 is sent
                await visitor.click(within(sheet, button('Grant to linux-1')));

                // Then - the engine's refusal is in the sheet, and linux-1 holds nothing
                await visitor.see(
                    within(sheet, content('did not confirm you are there', { exact: false })),
                );
                await visitor.gone(within(region('Granted to'), content('linux-1')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D30 a target that does not fit is held with the engine’s line', async () => {
        // Given - Claude Max's page over two-macs, linux-1 signed in by itself, api stopped, infra a server
        const result = await website.visit(
            '/?world=two-macs&page=passwords&item=pw-3&pace=600',
            async (visitor) => {
                // When - Grant… is pressed
                await visitor.click(button('Grant…'));
                const sheet = dialog('Grant Claude Max');

                // Then - each machine is held with why, none can be chosen, and nothing can be granted
                await visitor.see(within(sheet, content('It has its own sign-in')));
                await visitor.see(within(sheet, content('Start the machine first')));
                const held = disabled(field('linux-1'));
                const grant = disabled(button('Grant'));
                await visitor.see(within(sheet, held));
                await visitor.see(within(sheet, grant));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D32 a password added from a sign-in asks no kind and no value, and joins the list', async () => {
        // Given - the Passwords page over two-macs, gh signed in on this Mac
        const result = await website.visit('/?world=two-macs&page=passwords', async (visitor) => {
            await visitor.click(button('Add a password'));

            // When - its sentence is read, a label is given over the proposal, and it is added
            await visitor.see(within(SHEET, content('Kept as an OAuth token', { exact: false })));
            await visitor.gone(within(SHEET, button('fine-grained token')));
            await visitor.fill(within(SHEET, field('Label')), 'GitHub sign-in');
            await visitor.click(within(SHEET, button('Add password')));

            // Then - the sheet is gone, and the vault lists it
            await visitor.gone(SHEET);
            await visitor.see(
                within(region('Passwords'), button('GitHub sign-in', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test('D33 a typed password leaves in its request, and is drawn nowhere', async () => {
        // Given - the Add a password sheet over two-macs, on Cloudflare
        const result = await website.visit('/?world=two-macs&page=passwords', async (visitor) => {
            await visitor.click(button('Add a password'));
            await visitor.click(within(SHEET, button('Typed', { exact: false })));
            await visitor.click(within(SHEET, button('Cloudflare')));
            await visitor.fill(within(SHEET, field('Label')), 'Cloudflare deploy');

            // When - its value is typed, and it is added
            await visitor.fill(within(SHEET, field('Value')), 'good');
            await visitor.click(within(SHEET, button('Add password')));

            // Then - the sheet is gone, and the vault lists it
            await visitor.gone(SHEET);
            await visitor.see(
                within(region('Passwords'), button('Cloudflare deploy', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('good');
    });

    test('D34 a typed program offers its kinds, and none when it has one', async () => {
        // Given - the Add a password sheet over two-macs, the typed way
        const result = await website.visit('/?world=two-macs&page=passwords', async (visitor) => {
            await visitor.click(button('Add a password'));
            await visitor.click(within(SHEET, button('Typed', { exact: false })));

            // When - npm is picked, then Claude
            await visitor.click(within(SHEET, button('npm')));

            // Then - npm offers its kinds, with its value field; Claude, with one kind, offers none
            await visitor.see(within(SHEET, field('Value')));
            await visitor.see(within(SHEET, button('fine-grained token')));
            await visitor.click(within(SHEET, button('Claude')));
            await visitor.gone(within(SHEET, button('fine-grained token')));
        });
        expect(result.errors.text).toBe('');
    });

    test('D35 a refused label is said under the label, the sheet open and the value gone', async () => {
        // Given - npm typed under GitHub deploy, a label the vault already holds
        const result = await website.visit(
            '/?world=two-macs&page=passwords&pace=600',
            async (visitor) => {
                await visitor.click(button('Add a password'));
                await visitor.click(within(SHEET, button('Typed', { exact: false })));
                await visitor.click(within(SHEET, button('npm')));
                await visitor.fill(within(SHEET, field('Label')), 'GitHub deploy');
                await visitor.fill(within(SHEET, field('Value')), 'good');

                // When - it is added
                await visitor.click(within(SHEET, button('Add password')));

                // Then - the engine's refusal, and the sheet still open
                await visitor.see(
                    within(SHEET, content('a password is already labeled', { exact: false })),
                );
                await visitor.see(within(SHEET, field('Value')));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('good');
    });

    test('D36 an empty label is refused on the label, the sheet open and the value gone', async () => {
        // Given - npm typed with a value and no label
        const result = await website.visit(
            '/?world=two-macs&page=passwords&pace=600',
            async (visitor) => {
                await visitor.click(button('Add a password'));
                await visitor.click(within(SHEET, button('Typed', { exact: false })));
                await visitor.click(within(SHEET, button('npm')));
                await visitor.fill(within(SHEET, field('Value')), 'good');

                // When - it is added
                await visitor.click(within(SHEET, button('Add password')));

                // Then - the engine's refusal, and the sheet still open
                await visitor.see(
                    within(SHEET, content('a password needs a label', { exact: false })),
                );
                await visitor.see(within(SHEET, field('Value')));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('good');
    });

    test('D37 with no sign-in to keep on this Mac the way stays, says so, and Add stays off', async () => {
        // Given - the Add a password sheet over one-mac, no program signed in
        const result = await website.visit('/?world=one-mac&page=passwords', async (visitor) => {
            await visitor.click(button('Add a password'));

            // Then - the engine's line under the way, both ways drawn, and Add disabled
            await visitor.see(
                within(SHEET, content('Sign in to a program on this Mac first', { exact: false })),
            );
            await visitor.see(within(SHEET, button('Typed', { exact: false })));
            const add = disabled(button('Add password'));
            await visitor.see(within(SHEET, add));
        });
        expect(result.errors.text).toBe('');
    });
});
