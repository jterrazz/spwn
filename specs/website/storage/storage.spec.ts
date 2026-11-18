import { button, content, dialog, field, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('storage', () => {
    test.todo('D6 waits for the input of images.build');

    test('D8 the disk of a Mac is told by what Station holds and the engine verdict heads the page', async () => {
        // Given - Storage over two-macs
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - the page is read
            // Then - the engine's verdict heads it, and the disk says what station holds
            await visitor.see(heading('3.0 GB can be freed.'));
            await visitor.see(
                within(
                    region('Disk'),
                    content('Outside any machine, station holds 107 GB', { exact: false }),
                ),
            );
            await visitor.see(within(region('Disk'), content('Machine disks')));
        });
        expect(result.errors.text).toBe('');
    });

    test('D9 what nothing uses is freed by its names typed, and the verdict reads nothing to free', async () => {
        // Given - Storage over leftovers, an unused image and the disk of a deleted VM
        const subject = 'base-ubuntu@2026.10.20, old-vm';
        const result = await website.visit('/?world=leftovers&page=storage', async (visitor) => {
            await visitor.see(heading('12.0 GB can be freed.'));

            // When - Free 12 GB is pressed, and the names the engine asks are typed
            await visitor.click(within(region('Disk'), button('Free 12 GB')));
            const asked = dialog('Free 12 GB on macbook-pro?');
            await visitor.fill(within(asked, field(`Type ${subject} to confirm`)), subject);
            await visitor.click(within(asked, button('Free')));

            // Then - the verdict reads nothing to free, nothing is left to free, and what went is held
            await visitor.see(heading('Nothing to free.'));
            await visitor.gone(region('Can be freed'));
            await visitor.see(within(region('Done here'), content('old-vm')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('D9 waits for a kept journal on the wire, so Changes records what was freed');

    test('D10 choosing the other Mac at the top shows its own disk and its verdict', async () => {
        // Given - Storage over two-macs, where mac-mini's disk.read is recorded and has nothing to free
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            await visitor.see(heading('3.0 GB can be freed.'));

            // When - mac-mini is chosen
            await visitor.click(button('mac-mini'));

            // Then - its verdict heads the page, and the disk says what station holds outside any machine on it
            await visitor.see(heading('Nothing to free.'));
            await visitor.see(
                within(region('Disk'), content('station holds 84.0 GB', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test('D22 a machine of Storage opens its own Storage tab', async () => {
        // Given - Storage over two-macs, where linux-1 holds 50 GB
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - linux-1 is pressed under Machines
            await visitor.click(button('linux-1 1.5 GB of caches inside 50 GB'));

            // Then - its Storage tab opens, on the disk inside it
            await visitor.see(heading('linux-1 running'));
            await visitor.see(button('Free 1.5 GB'));
        });
        expect(result.errors.text).toBe('');
    });

    test('D22 each machine of Storage shows its size and the caches to free inside it', async () => {
        // Given - Storage over two-macs, where This Mac holds 34.8 GB of caches and linux-1 1.5 GB
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - Machines is read
            const machines = region('Machines');

            // Then - the Mac and linux-1 each say their caches and their size, api its size alone
            await visitor.see(
                within(
                    machines,
                    button('macbook-pro the Mac itself 34.8 GB of caches inside 499 GB'),
                ),
            );
            await visitor.see(within(machines, button('linux-1 1.5 GB of caches inside 50 GB')));
            await visitor.see(within(machines, button('api 50 GB')));
        });
        expect(result.errors.text).toBe('');
    });
});
