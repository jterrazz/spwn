import { button, content, field, heading, main, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const search = field('Search the catalog');

describe('catalog', () => {
    test('D4 an app the file lists reads its version rule in plain words', async () => {
        // Given - the window on gh's page over stories, an app This Mac's file lists
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=gh',
            async (visitor) => {
                // When - its Version is read
                const version = region('Version');

                // Then - the rule is the engine's sentence
                await visitor.see(within(version, content('takes every version')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "D4 waits for a recording of a night that left a version waiting (Line.Waiting), for each app's last move, and for Allow, which no call gives: it writes the owner's file",
    );

    test('D5 an item names the files that declare it and the machines that carry it', async () => {
        // Given - the window on gh's page over stories
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=gh',
            async (visitor) => {
                const rail = region('gh in the catalog');

                // When - its rail is read
                // Then - the files that declare it, and the machines that carry it
                await visitor.see(within(rail, content('Declared in')));
                await visitor.see(within(rail, button('studio-linux')));
                await visitor.see(within(rail, content('3 machines')));

                // When - the file is pressed
                await visitor.click(within(rail, button('studio-linux')));

                // Then - Machine files opens on it
                await visitor.see(heading('studio-linux.machine.yaml'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo("D5 waits for a package's source, which no call carries");

    test('D13 the search narrows the list as I type, and says when nothing matches', async () => {
        // Given - the window on Apps over two-macs
        const result = await website.visit(
            '/?world=two-macs&page=catalog-apps',
            async (visitor) => {
                await visitor.click(button('Everything', { exact: false }));
                await visitor.see(content('lazygit'));
                await visitor.see(content('ripgrep'));

                // When - a name is typed in the search
                await visitor.fill(search, 'lazygit');

                // Then - the list narrows to it
                await visitor.see(content('lazygit'));
                await visitor.gone(content('ripgrep'));

                // When - a word no package carries is typed
                await visitor.fill(search, 'zzzz');

                // Then - the list says nothing matches
                await visitor.see(content('Nothing matches.'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D13 by default only what the machines carry is listed, each with how many carry it', async () => {
        // Given - the window on Apps over stories
        const result = await website.visit('/?world=stories&page=catalog-apps', async (visitor) => {
            // When - the list is read
            // Then - a row says how many of the machines read carry the app, and Everything is offered
            await visitor.see(content('1 of 2 machines', { exact: false }));
            await visitor.see(button('Everything', { exact: false }));
        });
        expect(result.errors.text).toBe('');
    });

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

    test.todo(
        'D14 waits for a recording of an app installed per system, and of what is installed first for it and what it asks of the owner',
    );

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
                await visitor.see(within(applied, content('Can be undone')));
                await visitor.see(
                    within(changes, content("the agent user's login password", { exact: false })),
                );
                await visitor.see(
                    within(changes, content('/Library/Preferences/com.apple.loginwindow')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'D15 waits for a recording of a setting with options, their default and the owner value',
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

    test('D16 the folders an app keeps open Storage', async () => {
        // Given - the window on ghostty's page over stories
        const result = await website.visit(
            '/?world=stories&page=catalog-apps&item=ghostty',
            async (visitor) => {
                const kept = region('What it keeps on disk');
                await visitor.see(kept);

                // When - Storage is pressed
                await visitor.click(within(kept, button('Storage')));

                // Then - the Storage page opens
                await visitor.see(region('Disk'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('D16 waits for whether each setting an app comes with is on or off on a machine');

    test.todo("D17 waits for a call adding a package to a machine's file");

    test('D25 each nature of the catalog lists only its own packages', async () => {
        // Given - the window on Apps over stories
        const result = await website.visit('/?world=stories&page=catalog-apps', async (visitor) => {
            await visitor.see(region('Apps'));

            // When - Skills is pressed under Catalog
            await visitor.click(within(sidebar, button('Skills', { exact: false })));

            // Then - the page lists the agent skills alone
            await visitor.see(region('Skills'));
            await visitor.click(button('Everything', { exact: false }));
            await visitor.see(content('agent-browser-skill'));
            await visitor.gone(region('Apps'));

            // When - Settings is pressed
            await visitor.click(within(sidebar, button('Settings', { exact: false })));

            // Then - the page lists settings alone
            await visitor.see(region('Settings'));
            await visitor.gone(region('Skills'));
        });
        expect(result.errors.text).toBe('');
    });

    test('D25 the toolbar of a catalog page names its nature', async () => {
        // Given - the window on Skills over stories
        const result = await website.visit(
            '/?world=stories&page=catalog-skills',
            async (visitor) => {
                // When - the toolbar is read
                // Then - it reads Skills, and no Catalog page stands above it
                await visitor.see(within(main(), content('Skills')));
                await visitor.gone(content('Catalog ›'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D26 a category keeps only its packages, and each category shows its count', async () => {
        // Given - the window on Apps over stories
        const result = await website.visit('/?world=stories&page=catalog-apps', async (visitor) => {
            await visitor.click(button('Everything', { exact: false }));
            await visitor.see(button('Runtimes7'));
            await visitor.see(button('cocoapods', { exact: false }));
            await visitor.see(button('ripgrep', { exact: false }));

            // When - AI Agents is pressed
            await visitor.click(button('AI Agents8'));

            // Then - only its packages stay
            await visitor.see(button('claude', { exact: false }));
            await visitor.gone(button('cocoapods', { exact: false }));
            await visitor.gone(button('ripgrep', { exact: false }));
        });
        expect(result.errors.text).toBe('');
    });
});
