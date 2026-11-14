import { button, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('storage', () => {
    test.todo('D6 waits for the input of images.build');

    test.todo('D8 waits for what each kind holds, on disk.read');

    test.todo(
        'D9 window differs: Storage frees the caches (Free 34.8 GB) and lists no leftovers of deleted VMs, and its verdict is not given, where the row frees leftovers and reads "Nothing to clean."',
    );

    test.todo('D10 waits for a recording of the disk of the other Mac');

    test('D22 a machine of Storage opens its own Storage tab', async () => {
        // Given - Storage over two-macs, where linux-1 holds 50 GB
        const result = await website.visit('/?world=two-macs&page=storage', async (visitor) => {
            // When - linux-1 is pressed under Machines
            await visitor.click(button('linux-1 50 GB'));

            // Then - its Storage tab opens, on the disk inside it
            await visitor.see(heading('linux-1 Running'));
            await visitor.see(button('Free 1.5 GB'));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'D22 window differs: a machine of Machines shows its size alone, where the row shows the caches to free inside it too',
    );
});
