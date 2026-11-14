import { button, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('storage', () => {
    test.todo('D6 waits for the input of images.build');

    test.todo('D8 waits for what each kind holds, on disk.read');

    test.todo(
        "D9 waits for what nothing uses on disk.read, the leftovers of deleted VMs among it, and the page's verdict",
    );

    test.todo('D10 waits for a recording of the disk of the other Mac');

    test('D22 a machine of Storage opens its own Storage tab', async () => {
        // Given - Storage over two-macs, where linux-1 holds 50 GB
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - linux-1 is pressed under Machines
            await visitor.click(button('linux-1 1.5 GB of caches inside 50 GB'));

            // Then - its Storage tab opens, on the disk inside it
            await visitor.see(heading('linux-1 Running'));
            await visitor.see(button('Free 1.5 GB'));
        });
        expect(result.errors.text).toBe('');
    });

    test('D22 each machine of Storage shows its size and the caches to free inside it', async () => {
        // Given - Storage over two-macs, where This Mac holds 34.8 GB of caches and linux-1 1.5 GB
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - Machines is read
            const machines = region('Machines');

            // Then - the Mac and linux-1 each say their caches, linux-1 its size too, api none
            await visitor.see(
                within(
                    machines,
                    button('jterrazz-macbook-pro the Mac itself 34.8 GB of caches inside'),
                ),
            );
            await visitor.see(within(machines, button('linux-1 1.5 GB of caches inside 50 GB')));
            await visitor.see(within(machines, button('api 50 GB')));
        });
        expect(result.errors.text).toBe('');
    });
});
