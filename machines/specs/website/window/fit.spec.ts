import { button, content, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const PANEL = region('The engine on this Mac');

describe('fit', () => {
    test('E8 an engine older than this app is the verdict and its remedy, and Update brings the pages back', async () => {
        // Given - the proof build over an engine whose revision is below the oldest accepted
        const result = await website.visit(
            '/?world=two-macs&page=home&fit=engine-older&core=update',
            async (visitor) => {
                // When - the window opens on it
                await visitor.see(within(PANEL, heading('Update station here')));

                // Then - the install line, the verdict at the foot, and Update restores Home
                await visitor.see(
                    within(PANEL, content('curl -fsSL https://spwn.sh/install.sh | bash')),
                );
                await visitor.see(within(sidebar, content('Engine older than this app')));
                await visitor.click(within(PANEL, button('Update')));
                await visitor.see(heading('2 things need you.'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('E9 this app older than the engine is the verdict, with the window update as its remedy', async () => {
        // Given - the proof build over an engine whose revision is above the newest known, a window release out
        const result = await website.visit(
            '/?world=two-macs&page=home&fit=app-older&window=out',
            async (visitor) => {
                // When - the window opens on it
                await visitor.see(within(PANEL, heading('Update Spwn')));

                // Then - the verdict on the sidebar's Engine line, and the window's own Install and relaunch
                await visitor.see(within(sidebar, content('This app older than the engine')));
                await visitor.see(within(PANEL, button('Install and relaunch')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
