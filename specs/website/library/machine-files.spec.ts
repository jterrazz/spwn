import { button, content, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('machine files', () => {
    test('D1 a file lists the machines that follow it, with the engine verdict on them', async () => {
        // Given - the window on the file linux-dev over stories-file, which two machines follow and differ from
        const result = await website.visit(
            '/?world=stories-file&page=machine-files&file=linux-dev',
            async (visitor) => {
                // When - its followers are read
                const followers = region('Machines that follow it');

                // Then - the engine's sentence, and each machine with its count of differences
                await visitor.see(within(followers, content('2 machines differ.')));
                await visitor.see(within(followers, content('agent-7f')));
                await visitor.see(within(followers, content('2 differences')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D2 the name of the file a machine follows opens the file', async () => {
        // Given - the window on This Mac's Packages over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Packages',
            async (visitor) => {
                await visitor.see(content('lazygit'));

                // When - the name of its file is pressed in the head
                await visitor.click(
                    within(
                        heading('macbook-pro.', { exact: false }),
                        button('macbook-pro', { exact: true }),
                    ),
                );

                // Then - Machine files opens on that file
                await visitor.see(heading('macbook-pro.machine.yaml'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D2 a follower listed by its file opens its Packages tab', async () => {
        // Given - the window on the file linux-dev over stories-file
        const result = await website.visit(
            '/?world=stories-file&page=machine-files&file=linux-dev',
            async (visitor) => {
                // When - the machine agent-7f is pressed among its followers
                await visitor.click(
                    within(region('Machines that follow it'), button('agent-7f', { exact: false })),
                );

                // Then - the file's page is gone (its follower row bears the same name), and the head of that machine stands in its place
                await visitor.gone(region('Machines that follow it'));
                await visitor.see(heading('agent-7f Running'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D3 a file that makes a Mac offers no New machine', async () => {
        // Given - the window on the file macbook-pro over stories, which makes a Mac
        const result = await website.visit(
            '/?world=stories&page=machine-files&file=macbook-pro',
            async (visitor) => {
                const file = region('macbook-pro.machine.yaml');
                await visitor.see(within(file, content('a Mac')));

                // When - its head is read
                // Then - its head offers no New machine
                await visitor.gone(within(file, button('New machine')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'D3 waits for a recording of machines.new from a file that makes a workspace with an image, which names the file the VM follows',
    );

    test.todo(
        'D7 waits for a recording of a change applied from the window, and of the file it wrote',
    );

    test('D11 a file says what it makes, its system, size and the image it starts from', async () => {
        // Given - the window on the shipped file base-ubuntu over stories-file
        const result = await website.visit(
            '/?world=stories-file&page=machine-files&file=base-ubuntu',
            async (visitor) => {
                const file = region('base-ubuntu.machine.yaml');

                // When - its head is read
                // Then - its description, what it makes, its size and its image
                await visitor.see(within(file, content('A Linux workspace', { exact: false })));
                await visitor.see(within(file, content('a workspace')));
                await visitor.see(within(file, content('4 CPU · 8G · 20G')));
                await visitor.see(within(file, content('ghcr.io/cirruslabs/ubuntu:24.04')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D12 each chapter lists its lines grouped by the file that declares them', async () => {
        // Given - the window on the file macbook-pro over stories-file, which extends base
        const result = await website.visit(
            '/?world=stories-file&page=machine-files&file=macbook-pro',
            async (visitor) => {
                const puts = region('What it puts on a machine');
                const own = within(puts, region('macbook-pro'));
                const base = within(puts, region('base'));

                // When - Apps is read
                // Then - each line sits under the file that declares it
                await visitor.see(within(own, content('gh')));
                await visitor.see(within(base, content('homebrew')));

                // When - Settings is chosen
                await visitor.click(within(puts, button('Settings1')));

                // Then - its lines are listed under their file
                await visitor.see(within(own, content('git-shortcuts')));
                await visitor.gone(base);
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D12 every line of a chapter opens its catalog page', async () => {
        // Given - the Settings chapter of the file macbook-pro over stories-file
        const result = await website.visit(
            '/?world=stories-file&page=machine-files&file=macbook-pro',
            async (visitor) => {
                const puts = region('What it puts on a machine');
                await visitor.click(within(puts, button('Settings1')));

                // When - its line git-shortcuts is pressed
                await visitor.click(within(puts, button('git-shortcuts', { exact: false })));

                // Then - the catalog page of that setting opens
                await visitor.see(heading('git-shortcuts setting'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D18 every file is a tab under the caption of its shelf, one with a differing machine flagged', async () => {
        // Given - the window on Machine files over stories-file
        const result = await website.visit(
            '/?world=stories-file&page=machine-files',
            async (visitor) => {
                // When - the tabs are read
                // Then - three captions, the shelves' files under them, and the tab of a file with differing machines flagged
                await visitor.see(content('Your machines'));
                await visitor.see(content('Mixins'));
                await visitor.see(content('Shipped with station'));
                await visitor.see(button('base-macos'));
                await visitor.see(button('linux-dev 2 machines differ.'));

                // When - another tab is pressed
                await visitor.click(button('base-ubuntu'));

                // Then - the panel changes in place
                await visitor.see(heading('base-ubuntu.machine.yaml'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "D19 waits for a recording of a setting's options with their values and defaults, and the file that set each, on machinefile.explain",
    );

    test("D20 a Mac's own changes list what it added and left out", async () => {
        // Given - the window on the file macbook-pro over stories-file, whose Mac adds bat and leaves lazygit out
        const result = await website.visit(
            '/?world=stories-file&page=machine-files&file=macbook-pro',
            async (visitor) => {
                // When - the last panel is read
                const own = region("This Mac's own changes");

                // Then - the added app and the one left out
                await visitor.see(within(own, content('bat')));
                await visitor.see(within(own, content('an app the files declare, left out here')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D21 a mixin makes no machine and lists the files that extend it', async () => {
        // Given - the window on the mixin base over stories-file
        const result = await website.visit(
            '/?world=stories-file&page=machine-files&file=base',
            async (visitor) => {
                // When - its head and its users are read
                // Then - it makes no machine, and the file that extends it is listed
                await visitor.see(content('A mixin makes no machine by itself', { exact: false }));
                await visitor.see(within(region('Files that extend it'), content('macbook-pro')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo("D23 waits for a call that opens the owner's file to edit");

    test("D24 the file's text is read only, and a second tab shows the Mac's own changes file", async () => {
        // Given - the window on the file macbook-pro over stories-file
        const result = await website.visit(
            '/?world=stories-file&page=machine-files&file=macbook-pro',
            async (visitor) => {
                // When - YAML is chosen
                await visitor.click(within(region('macbook-pro.machine.yaml'), button('YAML')));

                // Then - the text of the file is shown, numbered
                await visitor.see(region('File text'));

                // When - the Mac's tab is pressed
                await visitor.click(button('This Mac'));

                // Then - its own changes file is shown
                await visitor.see(region('macbook-pro.local.yaml'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
