import {
    button,
    component,
    content,
    dialog,
    field,
    focused,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../engine/client';
import { EngineProvider } from '../../engine/react';
import { replayTransport } from '../../engine/replay';
import { recordingsOf } from '../../fixtures/recordings';
import { PAGES } from '../../pages/registry';
import { Window } from '../window';

const recordingFor = await recordingsOf('two-macs');

// hazard: press wraps a key longer than one letter in braces, so a chord is user-event's hold and release
// hazard: with its outer braces left to press, and a bracket is doubled to be read as itself.
const COMMAND_K = 'Control>}k{/Control';
const COMMAND_BACK = 'Control>}[[{/Control';

const find = dialog('Find');
const query = within(find, field('Find'));

function windowOnHome() {
    const client = new Client(replayTransport(recordingFor('two-macs')));
    return (
        <EngineProvider client={client}>
            <Window initial={{ page: 'home' }} pages={PAGES} />
        </EngineProvider>
    );
}

describe('Find', () => {
    test('⌘K opens it, a query narrows it, Enter opens the result', async () => {
        // Given - the window on Home over two-macs
        const result = await component.render(windowOnHome(), async (visitor) => {
            await visitor.see(content('npm sign-in expired'));

            // When - ⌘K is pressed, then linux-1 logs typed and Enter pressed
            await visitor.press(COMMAND_K);
            await visitor.see(focused(query));
            await visitor.fill(query, 'linux-1 logs');
            await visitor.press('Enter');

            // Then - the palette is gone and linux-1's Logs is open
            await visitor.gone(find);
            await visitor.see(region('What linux-1 prints'));
        });
        expect(result.errors.text).toBe('');
    });

    test('the choice starts on the first row as drawn, the machine before its tabs', async () => {
        // Given - the window on Home over two-macs
        const result = await component.render(windowOnHome(), async (visitor) => {
            await visitor.see(content('npm sign-in expired'));

            // When - Find is opened, linux typed and Enter pressed at once
            await visitor.press(COMMAND_K);
            await visitor.fill(query, 'linux');
            await visitor.press('Enter');
            await visitor.gone(find);
        });

        // Then - linux-1 itself opened, on its Overview, and none of its tabs
        expect(result.html.text).toMatch(/aria-label="Overview"[^>]*role="tabpanel"/u);
    });

    test('the arrows move the choice and wrap at both ends', async () => {
        // Given - the palette holding linux-1's Activity, Packages and Accounts, in the tab bar's order
        const result = await component.render(windowOnHome(), async (visitor) => {
            await visitor.click(within(navigation('Sidebar'), button('Find')));
            await visitor.fill(query, 'linux-1 › ac');

            // When - down three times, wrapping to Activity, then up twice, wrapping back through Accounts
            await visitor.press('ArrowDown');
            await visitor.press('ArrowDown');
            await visitor.press('ArrowDown');
            await visitor.press('ArrowUp');
            await visitor.press('ArrowUp');
            await visitor.press('Enter');
            await visitor.gone(find);
        });

        // Then - the choice was Packages, the tab now open
        expect(result.html.text).toMatch(/aria-label="Packages"[^>]*role="tabpanel"/u);
    });

    test('escape closes it and gives the keyboard back to its opener', async () => {
        // Given - the palette opened from the sidebar's Find
        const result = await component.render(windowOnHome(), async (visitor) => {
            const opener = within(navigation('Sidebar'), button('Find'));
            await visitor.click(opener);
            await visitor.see(focused(query));

            // When - Escape is pressed
            await visitor.press('Escape');

            // Then - the palette is gone and the keyboard is back on Find
            await visitor.gone(find);
            await visitor.see(focused(opener));
        });
        expect(result.errors.text).toBe('');
    });

    test('while a dialog is open, ⌘K and ⌘[ do nothing, so the draft stays', async () => {
        // Given - New machine opened from Home, a name typed in it
        const result = await component.render(windowOnHome(), async (visitor) => {
            await visitor.click(button('New machine'));
            const sheet = dialog('New machine on this Mac');
            await visitor.see(sheet);
            await visitor.fill(within(sheet, field('Name')), 'gate5');

            // When - ⌘K and ⌘[ are pressed inside it
            await visitor.press(COMMAND_K);
            await visitor.press(COMMAND_BACK);

            // Then - no palette, and the sheet still holds its draft
            await visitor.gone(find);
            await visitor.see(within(sheet, field('Name')));
        });
        expect(result.html.text).toContain('value="gate5"');
    });
});
