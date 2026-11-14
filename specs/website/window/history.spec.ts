import {
    button,
    content,
    disabled,
    enabled,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const HOME = '/?world=two-macs&page=home';
const sidebar = navigation('Sidebar');
const back = button('Back');
const forward = button('Forward');
const thisMac = within(sidebar, button('jterrazz-macbook-pro online'));

describe('history', () => {
    test('A4 back returns through the tab and the page I came from, then is disabled', async () => {
        // Given - the window on Home over two-macs, nothing behind it
        const result = await website.visit(HOME, async (visitor) => {
            await visitor.see(disabled(back));

            // When - This Mac is opened, its Ports tab chosen, then back is taken twice
            await visitor.click(thisMac);
            await visitor.click(content('Ports'));
            await visitor.see(region('What listens on this Mac'));
            await visitor.see(enabled(back));
            await visitor.click(back);
            await visitor.see(region('Its machines'));
            await visitor.click(back);
            await visitor.see(disabled(back));
            await visitor.see(content('npm sign-in expired'));
        });

        // Then - the window is back on Home, with nothing broken on the way
        expect(result.errors.text).toBe('');
    });

    test('A4 forward retraces what back undid, and is disabled at the last place', async () => {
        // Given - the window on Home over two-macs, nothing behind it nor ahead
        const result = await website.visit(HOME, async (visitor) => {
            await visitor.see(disabled(back));
            await visitor.see(disabled(forward));

            // When - This Mac is opened, back taken, then forward
            await visitor.click(thisMac);
            await visitor.see(heading('jterrazz-macbook-pro Online'));
            await visitor.see(disabled(forward));
            await visitor.click(back);
            await visitor.see(disabled(back));
            await visitor.see(enabled(forward));
            await visitor.click(forward);
            await visitor.see(heading('jterrazz-macbook-pro Online'));
            await visitor.see(enabled(back));
            await visitor.see(disabled(forward));
        });

        // Then - the window is on This Mac again, with nothing broken on the way
        expect(result.errors.text).toBe('');
    });

    test('A4 ⌘[ and ⌘] walk the history as the toolbar buttons do', async () => {
        // Given - the window on Home over two-macs
        const result = await website.visit(HOME, async (visitor) => {
            // When - This Mac is opened, then ⌘[ and ⌘] pressed
            await visitor.click(thisMac);
            await visitor.see(heading('jterrazz-macbook-pro Online'));
            await visitor.press('Control+BracketLeft');
            await visitor.see(content('npm sign-in expired'));
            await visitor.see(enabled(forward));
            await visitor.press('Control+BracketRight');
            await visitor.see(heading('jterrazz-macbook-pro Online'));
            await visitor.see(enabled(back));
        });

        // Then - the window is on This Mac again, with nothing broken on the way
        expect(result.errors.text).toBe('');
    });
});
