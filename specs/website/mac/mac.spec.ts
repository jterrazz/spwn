import { button, content, dialog, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');

describe('mac', () => {
    test('M1 a Mac reads its name, Online, its CPU, memory and disk', async () => {
        // Given - the window on This Mac's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Overview',
            async (visitor) => {
                // When - the page is read
                // Then - its name and state head it, and its three measures follow
                await visitor.see(heading('jterrazz-macbook-pro Online'));
                await visitor.see(content('This Mac · Engine running'));
                await visitor.see(content('34%'));
                await visitor.see(content('12.4GB'));
                await visitor.see(content('606GB'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'M1 window differs: the page does not say whether the Mac stays awake, nor until when',
    );

    test.todo('M2 waits for the choices awake.keep takes, worded');

    test('M3 the machines of a Mac are listed with their creator and their state', async () => {
        // Given - the window on This Mac's page over two-macs, which holds api and linux-1
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Overview',
            async (visitor) => {
                // When - Its machines is read
                const machines = region('Its machines');
                await visitor.see(machines);

                // Then - each machine names its creator and its state
                await visitor.see(within(machines, button('api by you Stopped')));
                await visitor.see(within(machines, button('linux-1 by you Running')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('M3 window differs: a machine of Its machines carries no purpose beside its creator');

    test('M4 New machine on a Mac opens the sheet with that Mac chosen', async () => {
        // Given - the window on the mini's page over two-macs, which is not This Mac
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-mac-mini',
            async (visitor) => {
                // When - its New machine is pressed
                await visitor.click(button('New machine on jterrazz-mac-mini'));

                // Then - the sheet opens on the mini
                await visitor.see(dialog('New machine on jterrazz-mac-mini'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M5 a Mac not seen reads Not seen, greyed, and answers nothing it does not know', async () => {
        // Given - the window on the mini's page over mini-offline, a Mac not seen for 12 min
        const result = await website.visit(
            '/?world=mini-offline&machine=jterrazz-mac-mini&tab=Overview',
            async (visitor) => {
                // When - the page and the sidebar are read
                // Then - the Mac reads Not seen in both, and its measures say no answer
                await visitor.see(heading('jterrazz-mac-mini Not seen'));
                await visitor.see(within(sidebar, button('jterrazz-mac-mini not seen')));
                await visitor.see(heading('No answer from the engine.'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'M5 window differs: the head draws no sentence saying nothing proves the Mac is down, and its VM reads Unknown where the row says Not seen',
    );

    test.todo('M5 waits for the values of an Activity page of a Mac not seen, said unknown');

    test('M6 a stopped engine reads Engine stopped in the sidebar and offers Start the engine', async () => {
        // Given - the window on Home over engine-stopped
        const result = await website.visit('/?world=engine-stopped&page=home', async (visitor) => {
            // When - the window opens
            // Then - the sidebar's foot says so, and the line of Needs you offers to start it
            await visitor.see(within(sidebar, content('Engine stopped')));
            await visitor.see(
                within(
                    region('Needs you'),
                    button('Needs a look Engine stopped', { exact: false }),
                ),
            );
            await visitor.see(within(region('Needs you'), button('Start the engine engine')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'M6 window differs: the verdict of Home is not given, where the row has the head say the engine stopped, in red, with when the page was last read',
    );

    test('M7 Free on one app frees its cache, and the bar and the verdict follow', async () => {
        // Given - the Storage tab of This Mac over two-macs, 34.8 GB of caches
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Storage',
            async (visitor) => {
                await visitor.see(within(region('Storage'), content('606 GB', { exact: false })));
                await visitor.see(within(region('By app'), button('Free go')));

                // When - Free is pressed on go
                await visitor.click(within(region('By app'), button('Free go')));

                // Then - what it held is freed, said under Done here, and the groups count less
                await visitor.see(within(region('Done here'), content('19.4 GB freed')));
                await visitor.see(within(region('Storage'), content('587 GB', { exact: false })));
                await visitor.see(button('Free 15.4 GB'));
                await visitor.gone(within(region('By app'), button('Free go')));
                await visitor.see(within(region('By app'), button('Erase… xcode')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'M7 window differs: the bar holds Misplaced, Can rebuild, Your data, Never touched and Free, where the row says system, apps, my data and caches',
    );

    test.todo('M7 waits for a kept journal that records a Free, read in Changes');

    test('M8 the Packages tab names the file the Mac follows, and says it matches', async () => {
        // Given - the Packages tab of This Mac over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Packages',
            async (visitor) => {
                // When - the tab is read
                // Then - it says Matches and names the file it follows
                await visitor.see(content('4 of 4 apps match', { exact: false }));
                const head = heading('Matches', { exact: false });
                await visitor.see(head);

                // When - the file's name is pressed
                await visitor.click(within(head, button('jterrazz-macbook-pro')));

                // Then - the file opens
                await visitor.see(region('What it puts on a machine'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'M9 waits for a recording of a Mac that lists what listens on it, each with its word',
    );

    test.todo(
        'M10 waits for a verb that picks one of several equal targets: github and claude both show the chip linux-1',
    );

    test('M11 Activity reads its four measures, its machines among the busiest, its heat and battery', async () => {
        // Given - the Activity tab of This Mac over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Activity',
            async (visitor) => {
                // When - the page is read
                // Then - the four measures, the busiest and the Mac's own facts are there
                await visitor.see(region('CPU'));
                await visitor.see(region('Memory'));
                await visitor.see(region('Disk'));
                await visitor.see(region('Network'));
                await visitor.see(within(region('This Mac'), content('96 % · charging')));
                await visitor.see(within(region('This Mac'), content('Nominal')));

                // When - its machine among the busiest is pressed
                await visitor.click(
                    within(
                        region('Busiest'),
                        button('linux-1 a machine of this Mac', { exact: false }),
                    ),
                );

                // Then - that machine opens
                await visitor.see(heading('linux-1 Running'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('M11 waits for a recording of enough samples for the four curves to be drawn');

    test('M12 the Packages tab of a Mac that matches its file says so, then lists its upgrades and its apps by layer', async () => {
        // Given - the Packages tab of This Mac over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Packages',
            async (visitor) => {
                // When - the tab is read
                // Then - the head matches, then upgrades, then the apps under each layer
                await visitor.see(content('4 of 4 apps match', { exact: false }));
                await visitor.see(region('Upgrades'));
                const apps = region('Apps');
                const base = within(apps, region('jterrazz-base'));
                const own = within(apps, region('jterrazz-macbook-pro'));
                await visitor.see(within(base, content('homebrew')));
                await visitor.see(within(own, content('lazygit')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo("M13 waits for a call reading the engine's lines and its setup runs");
});
