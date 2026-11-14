import { button, content, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const FILES = 'every machine file with its source and whether it is a mixin, on one read';
const FOLLOWERS = 'the machines that follow a file, and whether each matches it';

describe('machine files', () => {
    test.todo(`D1 waits for ${FOLLOWERS}`);

    test('D2 the name of the file a machine follows opens the file', async () => {
        // Given - the window on This Mac's Packages over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Packages',
            async (visitor) => {
                await visitor.see(content('lazygit'));

                // When - the name of its file is pressed in the head
                await visitor.click(
                    within(
                        heading('jterrazz-macbook-pro.', { exact: false }),
                        button('jterrazz-macbook-pro', { exact: true }),
                    ),
                );

                // Then - Machine files opens on that file
                await visitor.see(heading('jterrazz-macbook-pro.machine.yaml'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(`D2 waits for ${FOLLOWERS}`);

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

    test.todo('D7 lot 6');

    test.todo("D11 waits for the file's description and its VM's resources on machinefile.explain");

    test.todo('D12 lot 6');

    test.todo(`D18 waits for ${FILES}`);

    test.todo('D19 lot 6');

    test.todo(
        "D20 waits for the local file's additions, removals and values on machinefile.explain",
    );

    test.todo(`D21 waits for ${FILES}`);

    test.todo("D23 waits for a call that opens the owner's file to edit");

    test.todo("D24 waits for the file's text on machinefile.explain");
});
