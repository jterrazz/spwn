import {
    button,
    content,
    dialog,
    disabled,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');

describe('workspace', () => {
    test.todo(
        'W1 waits for the Tauri window, the only one that opens Terminal, and for the ssh command to be copied beside Open in Terminal',
    );

    test('W2 a workspace says what it is for, who created it and when', async () => {
        // Given - the window on linux-5 over creating, made by a session
        const result = await website.visit(
            '/?world=creating&machine=linux-5&tab=Overview',
            async (visitor) => {
                // When - Why it exists is read
                const why = region('Why it exists');
                await visitor.see(why);

                // Then - its purpose, its creator and its date are there
                await visitor.see(within(why, content('Lot shell, one block per package')));
                await visitor.see(within(why, content('4 Nov, by studio-os-fe')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('W2 waits for the hold on MachinesVM, which says whether it is safe to delete');

    test.todo('W3 waits for the hold on MachinesVM');

    test('W4 Stop reads Stopped everywhere, and the memory it held is free on Home', async () => {
        // Given - the window on Home over two-macs, where linux-1 runs and holds 8 GB of its Mac's 16
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            await visitor.see(within(region('Memory'), content('4 GB free')));
            await visitor.click(within(sidebar, button('linux-1 running')));
            await visitor.see(heading('linux-1 Running'));

            // When - Stop is pressed and confirmed
            await visitor.click(button('Stop'));
            await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));

            // Then - it reads Stopped, and Home counts its memory free
            await visitor.see(heading('linux-1 Stopped'));
            await visitor.see(within(sidebar, button('linux-1 stopped')));
            await visitor.click(within(sidebar, button('Home')));
            await visitor.see(within(region('Memory'), content('12 GB free')));
        });
        expect(result.errors.text).toBe('');
    });

    test('W4 Start brings a stopped workspace back', async () => {
        // Given - the window on api over two-macs, a workspace that is stopped
        const result = await website.visit('/?world=two-macs&machine=api', async (visitor) => {
            await visitor.see(heading('api Stopped'));

            // When - Start is pressed
            await visitor.click(button('Start'));

            // Then - it reads Running
            await visitor.see(heading('api Running'));
            await visitor.see(within(sidebar, button('api running')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'W4 waits for a recording of activity.read after a Stop, and of a Start after it on the same machine',
    );

    test('W5 a VM being created reads Creating, with its step', async () => {
        // Given - the window on linux-4 over creating, which is booting
        const result = await website.visit(
            '/?world=creating&machine=linux-4&tab=Overview',
            async (visitor) => {
                // When - the page is read
                // Then - it reads Creating and names the step it is at
                await visitor.see(heading('linux-4 Creating'));
                await visitor.see(content('Creating · boot 2 of 4'));
                await visitor.see(disabled(button('Open in Terminal')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('W5 waits for a recording of a failed creation, its step and its log');

    test.todo('W6 waits for an id on accounts.share');
});
