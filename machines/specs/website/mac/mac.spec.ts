import {
    button,
    content,
    dialog,
    field,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const rail = navigation('Rail');

describe('mac', () => {
    test('M1 a Mac reads its name, online, its CPU, memory and disk', async () => {
        // Given - the window on This Mac's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Overview',
            async (visitor) => {
                // When - the page is read
                // Then - its name and state head it, and its three measures follow
                await visitor.see(heading('macbook-pro online'));
                await visitor.see(content('This Mac · Engine running', { exact: false }));
                await visitor.see(content('34%'));
                await visitor.see(content('12.4GB'));
                await visitor.see(content('606GB'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M1 a Mac says whether it stays awake, and until when', async () => {
        // Given - This Mac over two-macs, which sleeps as usual, then over stories, kept awake until 18:00
        const sleeps = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Overview',
            async (visitor) => {
                // When - the page is read
                // Then - its line is the engine's sentence: it sleeps normally
                await visitor.see(content('This Mac · Engine running. Sleeps normally.'));
            },
        );
        const awake = await website.visit(
            '/?world=stories&machine=macbook-pro&tab=Overview',
            async (visitor) => {
                // When - the page is read
                // Then - its line says until when it stays awake, and that it does so lid closed
                await visitor.see(
                    content('This Mac · Engine running. Awake until 18:00, lid closed.'),
                );
            },
        );
        expect(sleeps.errors.text).toBe('');
        expect(awake.errors.text).toBe('');
    });

    test('M3 the machines of a Mac are listed with their creator and their state', async () => {
        // Given - the window on This Mac's page over two-macs, which holds api and linux-1
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Overview',
            async (visitor) => {
                // When - Its machines is read
                const machines = region('Its machines');
                await visitor.see(machines);

                // Then - each machine names its creator and its state
                await visitor.see(within(machines, button('api by you stopped')));
                await visitor.see(within(machines, button('linux-1 by you running')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M3 a machine of Its machines carries its purpose beside its creator', async () => {
        // Given - the window on This Mac's page over creating, where a session made linux-5 for a purpose
        const result = await website.visit(
            '/?world=creating&machine=macbook-pro&tab=Overview',
            async (visitor) => {
                // When - Its machines is read
                const machines = region('Its machines');
                await visitor.see(machines);

                // Then - linux-5 says what it is for and who made it, linux-4 who made it alone
                await visitor.see(
                    within(
                        machines,
                        button('linux-5 Lot shell, one block per package · by studio-os-fe', {
                            exact: false,
                        }),
                    ),
                );
                await visitor.see(within(machines, button('linux-4 by you', { exact: false })));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M4 New machine on a Mac opens the sheet with that Mac chosen', async () => {
        // Given - the window on the mini's page over two-macs, which is not This Mac
        const result = await website.visit('/?world=two-macs&machine=mac-mini', async (visitor) => {
            // When - its New machine is pressed
            await visitor.click(button('New machine on mac-mini'));

            // Then - the sheet opens on the mini
            await visitor.see(dialog('New machine on mac-mini'));
        });
        expect(result.errors.text).toBe('');
    });

    test('M5 a Mac not seen reads Not seen, greyed, and answers nothing it does not know', async () => {
        // Given - the window on the mini's page over mini-offline, a Mac not seen for 6 h
        const result = await website.visit(
            '/?world=mini-offline&machine=mac-mini&tab=Overview',
            async (visitor) => {
                // When - the page and the sidebar are read
                // Then - the Mac reads Not seen in both, and its head says what the engine says of it
                await visitor.see(heading('mac-mini not seen'));
                await visitor.see(within(sidebar, button('mac-mini not seen')));
                await visitor.see(heading('mac-mini has not been seen for 6 h.'));
                await visitor.see(
                    content('Nothing proves it is down: it may be asleep, or off your network.'),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M5 a VM of a Mac not seen reads Not seen, with the hold the engine gives it', async () => {
        // Given - the window on infra's page over mini-offline, infra being on the Mac not seen
        const result = await website.visit(
            '/?world=mini-offline&machine=infra&tab=Overview',
            async (visitor) => {
                // When - the page and the sidebar are read
                // Then - infra reads Not seen in both, and its hold says nothing is known of it
                await visitor.see(heading('infra not seen'));
                await visitor.see(within(sidebar, button('infra not seen')));
                await visitor.see(
                    content('Not seen: its Mac does not answer, so nothing says what holds it.'),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M5 the Activity of a Mac not seen draws no value, and says the Mac does not answer', async () => {
        // Given - the window on mac-mini's Activity over mini-offline, where its activity.read is refused as unreachable
        const result = await website.visit(
            '/?world=mini-offline&machine=mac-mini&tab=Activity',
            async (visitor) => {
                // When - the tab is read
                // Then - its head says the Mac does not answer, and no measure is drawn
                await visitor.see(heading('mac-mini does not answer.'));
                await visitor.gone(region('Memory'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M6 a stopped engine reads Engine stopped in the sidebar and offers Start the engine', async () => {
        // Given - the window on Home over engine-stopped
        const result = await website.visit('/?world=engine-stopped&page=home', async (visitor) => {
            // When - the window opens
            // Then - the sidebar's foot says so, and the line of Needs you offers to start it
            await visitor.see(within(rail, content('Engine stopped')));
            await visitor.see(
                within(region('Needs you'), button('Needs you Engine stopped', { exact: false })),
            );
            await visitor.see(within(region('Needs you'), button('Start the engine engine')));
        });
        expect(result.errors.text).toBe('');
    });

    test('M6 the head says the engine is stopped, with when the page was last read', async () => {
        // Given - the window on Home over engine-stopped
        const result = await website.visit('/?world=engine-stopped&page=home', async (visitor) => {
            // When - the window opens
            // Then - the engine's verdict heads the page in red, its line says what stops, and when it was read
            await visitor.see(heading('The engine is stopped.'));
            await visitor.see(
                content('Your VMs do not run and nothing is shared until it starts.', {
                    exact: false,
                }),
            );
            await visitor.see(content('Last read at', { exact: false }));
        });
        expect(result.errors.text).toBe('');
    });

    test('M7 Free on one app frees its cache, and the bar and the verdict follow', async () => {
        // Given - the Storage tab of This Mac over two-macs, 34.8 GB of caches
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Storage',
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

    test('M7 the bar of a Mac holds what the disk holds, by the kinds the engine gives', async () => {
        // Given - the Storage tab of This Mac over two-macs, whose disk.read gives system, apps, data, caches, machines and free
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Storage',
            async (visitor) => {
                // When - the tab is read
                const storage = region('Storage');

                // Then - each kind is named with its bytes, and the Mac says what it holds itself
                for (const kind of ['System', 'Your data', 'Caches', 'Machines and images']) {
                    await visitor.see(within(storage, content(kind)));
                }
                for (const size of ['21.4 GB', '443 GB', '107 GB']) {
                    await visitor.see(within(storage, content(size)));
                }
                await visitor.see(within(storage, content('This Mac itself holds 499 GB.')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M7 a cache freed is recorded in the History, by me', async () => {
        // Given - the Storage tab of This Mac over two-macs, whose History holds no day but last night
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Storage',
            async (visitor) => {
                // When - Free is pressed on go, then the Inbox opened
                await visitor.click(within(region('By app'), button('Free go')));
                await visitor.see(within(region('Done here'), content('19.4 GB freed')));
                await visitor.click(within(rail, button('Inbox', { exact: false })));

                // Then - a Today group holds the engine's line of the free, its time first
                await visitor.see(within(region('Today'), button('10:00 go freed by you')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M8 the Packages tab names the file the Mac follows, and says it matches', async () => {
        // Given - the Packages tab of This Mac over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Packages',
            async (visitor) => {
                // When - the tab is read
                // Then - it says Matches and names the file it follows
                await visitor.see(content('4 of 4 apps match', { exact: false }));
                const head = heading('Matches', { exact: false });
                await visitor.see(head);

                // When - the file's name is pressed
                await visitor.click(within(head, button('macbook-pro')));

                // Then - the file opens
                await visitor.see(region('What it puts on a machine'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M9 a Mac lists what listens on it, each port with the word of its reach', async () => {
        // Given - the window over stories, where macbook-pro lists ControlCenter on :5000, reached outside Tailscale, and mac-mini screensharingd on :5900, private
        const exposed = await website.visit(
            '/?world=stories&machine=macbook-pro&tab=Ports',
            async (visitor) => {
                // When - the list is read
                // Then - the exposed port is there with its word
                const listening = region('What listens on this Mac');
                await visitor.see(within(listening, content('ControlCenter', { exact: false })));
                await visitor.see(within(listening, content('Outside Tailscale')));
            },
        );
        const hidden = await website.visit(
            '/?world=stories&machine=mac-mini&tab=Ports',
            async (visitor) => {
                const listening = region('What listens on this Mac');
                await visitor.see(within(listening, content('screensharingd', { exact: false })));
                await visitor.see(within(listening, content('Private')));
            },
        );
        expect(exposed.errors.text).toBe('');
        expect(hidden.errors.text).toBe('');
    });

    test("M10 a Mac's Sign-ins tab lists each program with its state and the action the engine names", async () => {
        // Given - the Sign-ins tab of This Mac over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Sign-ins',
            async (visitor) => {
                // When - its rows are read
                const rows = region('Sign-ins');
                await visitor.see(rows);

                // Then - each program is a row, and a signed-in one offers its sign-out
                await visitor.see(within(rows, content('gh')));
                await visitor.see(within(rows, button('Sign out claude')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('M12 the Packages tab of a Mac that matches its file says so, then lists its upgrades and its apps by file', async () => {
        // Given - the Packages tab of This Mac over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Packages',
            async (visitor) => {
                // When - the tab is read
                // Then - the head matches, then upgrades, then the apps under each file
                await visitor.see(content('4 of 4 apps match', { exact: false }));
                await visitor.see(region('Upgrades'));
                const apps = region('Apps');
                const base = within(apps, region('base'));
                const own = within(apps, region('macbook-pro'));
                await visitor.see(within(base, content('homebrew')));
                await visitor.see(within(own, content('lazygit')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("M13 a Mac's log reads the lines station wrote there, a withheld one as the engine sent it, and the filter narrows them", async () => {
        // Given - the Activity of This Mac over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Activity',
            async (visitor) => {
                // When - its log is read
                const log = region('What Machines did on this Mac');
                await visitor.see(within(log, content('nightly run done')));

                // Then - every line is the engine's, the one it withheld as it came
                await visitor.see(within(log, content('engine started')));
                await visitor.see(within(log, content('ping https://withheld@example.com/x ok')));

                // When - a filter is typed
                await visitor.fill(within(log, field('Filter the lines')), 'nightly');

                // Then - the lines that hold it stay, counted out of the whole
                await visitor.see(within(log, content('2 of 4 lines', { exact: false })));
                await visitor.see(within(log, content('nightly run started')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("M13 a Mac whose engine wrote no line says the engine's sentence for it", async () => {
        // Given - the Activity of This Mac over bare-mac, whose engine has written no line
        const result = await website.visit(
            '/?world=bare-mac&machine=macbook-pro&tab=Activity',
            async (visitor) => {
                // When - its log is read
                // Then - it says so in the engine's words
                await visitor.see(
                    within(
                        region('What Machines did on this Mac'),
                        content('The engine has written no line here yet.'),
                    ),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });
});
