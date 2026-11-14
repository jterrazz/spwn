import { button, content, dialog, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');

describe('workspace', () => {
    test.todo('W1 lot 6');

    test.todo('W2 lot 6');

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

    test.todo('W5 lot 6');

    test.todo('W6 waits for an id on accounts.share');
});
