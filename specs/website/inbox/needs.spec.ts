import { button, content, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const needs = region('Needs you');

describe('needs', () => {
    test('N1 a line settled by its verb leaves, and the sidebar counts one less', async () => {
        // Given - the window on Home over two-macs, nine lines waiting, one of them a mise update
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            await visitor.see(within(sidebar, button('Inbox 9')));
            await visitor.see(heading('Needs you 9'));
            await visitor.see(within(needs, content('1 mise update waiting')));

            // When - its verb, Update mise, is pressed
            await visitor.click(within(needs, button('Update mise')));

            // Then - the line has left, and the sidebar and the panel count eight
            await visitor.gone(within(needs, button('Update mise')));
            await visitor.see(within(sidebar, button('Inbox 8')));
            await visitor.see(heading('Needs you 8'));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('N1 waits for a verdict and its line on overview.read');

    test.todo("N2 waits for a need's upkeep word on OverviewNeed");

    test.todo('N3 waits for a recording of an exposed port and the Review verb of its line');

    test.todo('N4 lot 6');

    test.todo("N5 waits for each line's version rule on machinefile.drift");

    test.todo(
        'N6 waits for a recording of an idle VM among the needs, and a kept journal for Changes',
    );

    test.todo('N7 lot 6');

    test.todo('N8 lot 6');

    test.todo('N9 waits for a kept journal, read by day');

    test.todo('N10 lot 6');

    test.todo('N11 lot 6');
});
