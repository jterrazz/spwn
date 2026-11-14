import { content, dialog, field, focused, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const find = dialog('Find');
const query = within(find, field('Find'));

describe('find', () => {
    test('A1 ⌘K, a query and Enter open the machine tab it names', async () => {
        // Given - the window on Home over two-macs
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            await visitor.see(content('npm sign-in expired'));

            // When - ⌘K is pressed, then linux-1 activity typed and Enter pressed
            await visitor.press('Control+k');
            await visitor.see(focused(query));
            await visitor.fill(query, 'linux-1 activity');
            await visitor.press('Enter');

            // Then - the palette is gone and linux-1's Activity is open, its log at its foot
            await visitor.gone(find);
            await visitor.see(region('What linux-1 prints'));
        });
        expect(result.errors.text).toBe('');
    });

    test('A1 two letters and Enter open the machine from the Inbox', async () => {
        // Given - the window on the Inbox over two-macs
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            await visitor.see(content('npm sign-in expired'));

            // When - ⌘K is pressed, then li typed and Enter pressed
            await visitor.press('Control+k');
            await visitor.fill(query, 'li');
            await visitor.press('Enter');

            // Then - linux-1 is open
            await visitor.gone(find);
            await visitor.see(heading('linux-1 Running'));
        });
        expect(result.errors.text).toBe('');
    });
});
