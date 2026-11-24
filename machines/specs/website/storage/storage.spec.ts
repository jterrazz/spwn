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

const THIS_MAC = region('macbook-pro');
const MINI = region('mac-mini');
const thisDisk = within(THIS_MAC, region('Disk'));
const miniDisk = within(MINI, region('Disk'));

describe('storage', () => {
    test('D6 an image being built shows its four steps', async () => {
        // Given - Storage over creating, an image held at its second step
        const result = await website.visit('/?world=creating&page=storage', async (visitor) => {
            // When - its images have been read
            const images = region('Images');
            await visitor.see(within(images, content('Packages · Step 2 of 4')));

            // Then - its other steps stand in the engine's words
            await visitor.see(within(images, content('Base system')));
            await visitor.see(within(images, content('Settings')));
            await visitor.see(within(images, content('Seal')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'D6 waits for images.build to answer in the world build, which as an unreleased station refuses it with image.no-linux-binary (apps/engine/images/calls/binary.go:26)',
    );

    test('D8 the disk of a Mac is told by what Machines holds on it and the engine verdict heads the page', async () => {
        // Given - Storage over two-macs
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - the page is read
            // Then - the engine's verdict heads it, and the disk says what station holds
            await visitor.see(within(THIS_MAC, heading('3.0 GB can be freed.')));
            await visitor.see(
                within(
                    thisDisk,
                    content('Outside any machine, Machines holds 107 GB', { exact: false }),
                ),
            );
            await visitor.see(within(thisDisk, content('Machine disks')));
        });
        expect(result.errors.text).toBe('');
    });

    test('D9 what nothing uses is freed by its names typed, and the verdict reads nothing to free', async () => {
        // Given - Storage over leftovers, an unused image and the disk of a deleted VM
        const subject = 'base-ubuntu@2026.10.20, old-vm';
        const result = await website.visit('/?world=leftovers&page=storage', async (visitor) => {
            await visitor.see(within(THIS_MAC, heading('12.0 GB can be freed.')));

            // When - Free 12 GB is pressed, and the names the engine asks are typed
            await visitor.click(within(thisDisk, button('Free 12 GB')));
            const asked = dialog('Free 12 GB on macbook-pro?');
            await visitor.fill(within(asked, field(`Type ${subject} to confirm`)), subject);
            await visitor.click(within(asked, button('Free')));

            // Then - the verdict reads nothing to free, nothing is left to free, and what went is held
            await visitor.see(within(THIS_MAC, heading('Nothing to free.')));
            await visitor.gone(within(THIS_MAC, region('Can be freed')));
            const done = within(THIS_MAC, region('Done here'));
            await visitor.see(within(done, content('old-vm')));
        });
        expect(result.errors.text).toBe('');
    });

    test('D9 what nothing uses, once freed, is recorded in the History, one line per name', async () => {
        // Given - Storage over leftovers, whose History holds no day
        const subject = 'base-ubuntu@2026.10.20, old-vm';
        const result = await website.visit('/?world=leftovers&page=storage', async (visitor) => {
            // When - Free 12 GB is pressed and confirmed with the names, then the Inbox opened
            await visitor.click(within(thisDisk, button('Free 12 GB')));
            const asked = dialog('Free 12 GB on macbook-pro?');
            await visitor.fill(within(asked, field(`Type ${subject} to confirm`)), subject);
            await visitor.click(within(asked, button('Free')));
            await visitor.see(within(THIS_MAC, heading('Nothing to free.')));
            await visitor.click(within(navigation('Rail'), button('Inbox', { exact: false })));

            // Then - a Today group holds the engine's line of each name it freed
            const today = region('Today');
            await visitor.see(within(today, content('base-ubuntu@2026.10.20 freed')));
            await visitor.see(within(today, content('old-vm freed')));
        });
        expect(result.errors.text).toBe('');
    });

    test('D10 every Mac has its section, the other Mac its own disk and its verdict under This Mac', async () => {
        // Given - Storage over two-macs, where mac-mini's disk.read is recorded and has nothing to free
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - the page is read
            await visitor.see(within(THIS_MAC, heading('3.0 GB can be freed.')));

            // Then - mac-mini's verdict heads its own section, its disk says what station holds outside any
            // Then - machine on it, and no Mac is a choice to make
            await visitor.see(within(MINI, heading('Nothing to free.')));
            await visitor.see(
                within(miniDisk, content('Machines holds 84.0 GB', { exact: false })),
            );
            await visitor.gone(button('mac-mini'));
        });
        expect(result.errors.text).toBe('');
    });

    test('D22 a machine of Storage opens its own Storage tab', async () => {
        // Given - Storage over two-macs, where linux-1 holds 50 GB
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - linux-1 is pressed under This Mac's Machines
            await visitor.click(within(THIS_MAC, button('linux-1 1.5 GB of caches inside 50 GB')));

            // Then - its Storage tab opens, on the disk inside it
            await visitor.see(heading('linux-1 running'));
            await visitor.see(button('Free 1.5 GB'));
        });
        expect(result.errors.text).toBe('');
    });

    test('D22 each machine of Storage shows its size and the caches to free inside it', async () => {
        // Given - Storage over two-macs, where This Mac holds 34.8 GB of caches and linux-1 1.5 GB
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - This Mac's Machines is read
            const machines = within(THIS_MAC, region('Machines'));

            // Then - the Mac says its caches and its size as a line that opens nothing, its disk being its
            // Then - section's; linux-1 says its caches and its size, api its size alone, each opening its tab
            await visitor.see(within(machines, content('the Mac itself')));
            await visitor.see(within(machines, content('34.8 GB of caches inside')));
            await visitor.gone(within(machines, button('macbook-pro', { exact: false })));
            await visitor.see(within(machines, button('linux-1 1.5 GB of caches inside 50 GB')));
            await visitor.see(within(machines, button('api 50 GB')));
        });
        expect(result.errors.text).toBe('');
    });

    test('D38 a kind that holds nothing on a machine is neither drawn nor listed', async () => {
        // Given - linux-1's Storage tab over stories, whose disk.read gives system, apps and caches at 0 bytes
        const result = await website.visit(
            '/?world=stories&machine=linux-1&tab=Storage',
            async (visitor) => {
                // When - the tab is read
                const storage = region('Disk');
                await visitor.see(within(storage, content('Your data')));

                // Then - only what holds bytes is listed, and free last
                await visitor.see(within(storage, content('Free')));
                for (const kind of ['System', 'Apps', 'Caches']) {
                    await visitor.gone(within(storage, content(kind)));
                }
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('D38 a kind of Machines that holds nothing on a Mac is neither drawn nor listed', async () => {
        // Given - Storage over two-macs, whose disk.read gives leftovers and logs at 0 bytes
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - the disk is read
            const disk = thisDisk;
            await visitor.see(within(disk, content('Machine disks')));

            // Then - leftovers and logs are gone from it, the others listed
            await visitor.see(within(disk, content('Images')));
            await visitor.gone(within(disk, content('Leftovers')));
            await visitor.gone(within(disk, content("Machines' logs")));
        });
        expect(result.errors.text).toBe('');
    });
});
