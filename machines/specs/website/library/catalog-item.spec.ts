import { button, content, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('catalog item', () => {
    test('D14 an app page reads how it is installed and the commands it puts on the path', async () => {
        // Given - the window on node's page over stories
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=node',
            async (visitor) => {
                // When - the page is read
                const installed = region('How it is installed');
                await visitor.see(installed);

                // Then - the manager and what it runs, and the commands put on the path
                await visitor.see(within(installed, content('mise')));
                await visitor.see(within(installed, content('Puts on your path')));
                await visitor.see(within(installed, content('npx')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D14 an app installed one way per system reads each, with its manager and what it runs', async () => {
        // Given - the window on postgres's page over stories, a brew formula on macOS, an apt package on Ubuntu
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=postgres',
            async (visitor) => {
                // When - the page is read
                const installed = region('How it is installed');
                await visitor.see(installed);

                // Then - one line per way, the system, the manager and what it fetches, all as the engine sent them
                await visitor.see(within(installed, content('macOS')));
                await visitor.see(within(installed, content('brew')));
                await visitor.see(within(installed, content('postgresql@18')));
                await visitor.see(within(installed, content('Ubuntu')));
                await visitor.see(within(installed, content('apt')));
                await visitor.see(within(installed, content('postgresql')));
                await visitor.see(within(installed, content('psql')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D14 an app page names what is installed first for it, and opens it', async () => {
        // Given - the window on codex's page over stories, an app bun installs
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=codex',
            async (visitor) => {
                const installed = region('How it is installed');
                await visitor.see(within(installed, content('Installs first')));

                // When - bun is pressed
                await visitor.click(within(installed, button('bun', { exact: true })));

                // Then - bun's page opens
                await visitor.see(heading('bun', { exact: false }));
                await visitor.gone(heading('codex', { exact: false }));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D14 an app page says what it asks of me', async () => {
        // Given - the window on xcode's page over stories, an app whose install needs its license accepted
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=xcode',
            async (visitor) => {
                // When - the page is read
                const installed = region('How it is installed');

                // Then - the need in the engine's word
                await visitor.see(within(installed, content('Asks you for')));
                await visitor.see(within(installed, content('a license accepted')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D15 a setting page reads each file it writes, and that it can be undone', async () => {
        // Given - the window on autologin's page over stories, a setting that writes two files
        const result = await website.visit(
            '/?world=stories&page=catalog-settings&item=autologin',
            async (visitor) => {
                // When - the page is read
                const applied = region('How it is applied');
                const changes = region('What it changes');
                await visitor.see(heading('autologin setting'));

                // Then - it can be undone, and each file says which part is station's
                await visitor.see(within(applied, content('can be undone')));
                await visitor.see(
                    within(
                        changes,
                        content('the login password of the user who logs in', { exact: false }),
                    ),
                );
                await visitor.see(
                    within(changes, content('/Library/Preferences/com.apple.loginwindow')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "D15 waits for an option's default and its values on CatalogInput: settings.list gives an option its name, its label and a numbered Kind alone",
    );

    test('D16 an app page lists the settings it comes with and the folders it keeps, with a link to Storage', async () => {
        // Given - the window on ghostty's page over stories, an app that proposes a setting and keeps data on disk
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=ghostty',
            async (visitor) => {
                const comesWith = region('Settings it comes with');
                const kept = region('What it keeps on disk');

                // When - the page is read
                await visitor.see(within(comesWith, button('ghostty-dotfiles', { exact: false })));
                await visitor.see(within(kept, content('~/Library/WebKit/com.mitchellh.ghostty')));

                // Then - the setting, a row drawn with its kind and its description, opens its page
                await visitor.click(
                    within(comesWith, button('ghostty-dotfiles', { exact: false })),
                );
                await visitor.see(heading('ghostty-dotfiles setting'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D16 each setting an app comes with says whether it is applied on This Mac', async () => {
        // Given - the window on ghostty's page over stories, whose ghostty-dotfiles This Mac's settings.list reads not applied
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=ghostty',
            async (visitor) => {
                // When - the settings it comes with are read
                const row = within(
                    region('Settings it comes with'),
                    button('ghostty-dotfiles', { exact: false }),
                );

                // Then - the row carries the setting's state
                await visitor.see(within(row, content('not applied')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "D16 waits for a setting's state on each other machine typed on settings.matrix, whose SettingsCell.State is an open string",
    );

    test('D16 the folders an app keeps open Storage', async () => {
        // Given - the window on ghostty's page over stories
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=ghostty',
            async (visitor) => {
                const kept = region('What it keeps on disk');
                await visitor.see(kept);

                // When - Storage is pressed
                await visitor.click(within(kept, button('Storage')));

                // Then - the Storage page opens, This Mac's disk in its section
                await visitor.see(within(region('macbook-pro'), region('Disk')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
