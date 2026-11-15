import { button, component, content, dialog, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('two-macs', 'bare-mac');

function passwordsOver(world: string, item?: string) {
    return pageOver(recordingFor(world), { item, page: 'passwords' });
}

describe('passwords', () => {
    test("this Mac's vault lists each password with what it is and where it is granted", async () => {
        // Given - the Passwords page over two-macs
        const result = await component.render(passwordsOver('two-macs'), async (visitor) => {
            // When - vault.list has answered
            await visitor.see(heading('Passwords on macbook-pro'));

            // Then - the engine's verdict, and a row a password, with its kind and its grants counted
            await visitor.see(content('3 kept, 1 granted', { exact: false }));
            const passwords = region('Passwords');
            await visitor.see(within(passwords, button('GitHub deploy', { exact: false })));
            await visitor.see(within(passwords, content('1 machine')));
            await visitor.see(within(passwords, content('fine-grained token')));
        });
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-passwords.aria.yaml'));
    });

    test('an empty vault says so in the engine’s words', async () => {
        // Given - the Passwords page over bare-mac, whose vault holds nothing
        const result = await component.render(passwordsOver('bare-mac'), async (visitor) => {
            // When - vault.list has answered
            // Then - the verdict alone, and an empty list
            await visitor.see(content('No password kept yet', { exact: false }));
        });
        expect(result.errors.text).toBe('');
    });

    test('a password opens on its facts and its grants, its value withheld', async () => {
        // Given - the Passwords page over two-macs
        const result = await component.render(passwordsOver('two-macs'), async (visitor) => {
            // When - npm publish is opened
            await visitor.click(
                within(region('Passwords'), button('npm publish', { exact: false })),
            );

            // Then - its facts, its value withheld, and its one grant
            await visitor.see(heading('npm publish'));
            await visitor.see(within(region('This password'), content('Withheld')));
            await visitor.see(within(region('Granted to'), content('linux-1')));
        });
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-pw-2.aria.yaml'));
    });

    test('removing a password asks first, saying how many grants end, then lists what is left', async () => {
        // Given - npm publish's page over two-macs, granted to linux-1
        const result = await component.render(
            passwordsOver('two-macs', 'pw-2'),
            async (visitor) => {
                // When - Remove password is clicked and confirmed
                await visitor.click(button('Remove password'));
                const asked = dialog('Remove npm publish?');
                await visitor.see(asked);
                await visitor.see(within(asked, content('Its grants end: 1 machine.')));
                await visitor.click(within(asked, button('Remove')));

                // Then - the vault is listed again, without it
                await visitor.see(heading('Passwords on macbook-pro'));
                await visitor.gone(
                    within(region('Passwords'), button('npm publish', { exact: false })),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('removing a grant asks first, then the password is granted nowhere', async () => {
        // Given - npm publish's page over two-macs, granted to linux-1
        const result = await component.render(
            passwordsOver('two-macs', 'pw-2'),
            async (visitor) => {
                // When - its grant to linux-1 is removed
                await visitor.click(button('Remove npm publish from linux-1'));
                const asked = dialog('Remove npm publish from linux-1?');
                await visitor.click(within(asked, button('Remove')));

                // Then - the grant's row is gone from its panel
                await visitor.gone(
                    within(region('Granted to'), button('Remove npm publish from linux-1')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });
});
