import {
    button,
    content,
    dialog,
    disabled,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('passwords', () => {
    test("D27 the sidebar's Passwords lists this Mac's vault under the engine's verdict", async () => {
        // Given - the window on Home over two-macs
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            // When - Passwords is pressed in the sidebar
            await visitor.click(within(navigation('Sidebar'), button('Passwords')));

            // Then - the page names this Mac, and each password says what it is and where it is granted
            await visitor.see(heading('Passwords on jterrazz-macbook-pro'));
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
                const asked = dialog('Remove npm publish? Its grants end: 1 machine.');
                await visitor.click(within(asked, button('Remove')));

                // Then - the vault is listed again without it
                await visitor.see(heading('Passwords on jterrazz-macbook-pro'));
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
                await visitor.click(within(sheet, button('linux-1', { exact: false })));

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

                // Then - each machine is held with why, and nothing can be granted
                await visitor.see(within(sheet, content('It has its own sign-in')));
                await visitor.see(within(sheet, content('Start the machine first')));
                const held = disabled(button('linux-1', { exact: false }));
                const grant = disabled(button('Grant'));
                await visitor.see(within(sheet, held));
                await visitor.see(within(sheet, grant));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
