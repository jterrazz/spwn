import { button, component, content, dialog, field, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { NO_OPENER } from '../../../infrastructure/outbound/opener-tauri/tauri-opener';

const recordingFor = await recordingsOf('sign-ins', 'two-macs');

// value: each frame of a watch held this long, so a step is on screen before the next one replaces it.
const PACE = { gestureFrame: 600 };

function signInsOver(world: string) {
    return pageOver(
        recordingFor(world),
        { machine: 'linux-1', page: 'machine', tab: 'Sign-ins' },
        NO_OPENER,
        PACE,
    );
}

const PASTE = 'Sign in to claude on linux-1';
const DEVICE_CODE = 'Sign in to gh on linux-1';

describe('login sheet', () => {
    test('a device code is shown with its copy, and the sheet closes once the engine signs in', async () => {
        // Given - linux-1's Sign-ins over two-macs, gh signed out
        const result = await component.render(signInsOver('two-macs'), async (visitor) => {
            // When - Sign in of gh is clicked
            await visitor.click(within(region('Sign-ins'), button('Sign in gh')));

            // Then - the engine's step: its heading and detail, its status line, its code to copy, its site to open, then the sheet goes
            const sheet = dialog(DEVICE_CODE);
            await visitor.see(within(sheet, content('Enter the code')));
            await visitor.see(
                within(sheet, content('Type it on the page that opened', { exact: false })),
            );
            await visitor.see(
                within(sheet, content('Waiting for you to enter the', { exact: false })),
            );
            await visitor.see(within(sheet, content('WDJB-MJHT')));
            await visitor.see(within(sheet, button('Copy code')));
            await visitor.see(within(sheet, button('Open github.com')));
            await visitor.see(within(sheet, content('05:00')));
            await visitor.gone(sheet);
        });
        expect(result.errors.text).toBe('');
    });

    test('a paste-back code goes through a password field the page never holds', async () => {
        // Given - claude's sign-in on linux-1 over sign-ins, asking its code back
        const result = await component.render(signInsOver('sign-ins'), async (visitor) => {
            await visitor.click(within(region('Sign-ins'), button('Sign in claude')));
            const sheet = dialog(PASTE);
            await visitor.see(within(sheet, content('Paste the code')));
            await visitor.see(
                within(sheet, content('then paste the code it gives you', { exact: false })),
            );

            // When - the code is pasted and sent
            await visitor.fill(within(sheet, field('Code from claude.com')), 'good');
            await visitor.click(within(sheet, button('Continue')));

            // Then - the field is empty again, and the code is nowhere on the page
            await visitor.see(within(sheet, field('Code from claude.com')));
        });
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('good');
        expect(result.html.text).toContain('type="password"');
    });

    test('a refused code is said, and the field is left empty for another', async () => {
        // Given - claude's sign-in on linux-1 over sign-ins
        const result = await component.render(signInsOver('sign-ins'), async (visitor) => {
            await visitor.click(within(region('Sign-ins'), button('Sign in claude')));
            const sheet = dialog(PASTE);

            // When - a code the provider refuses is sent
            await visitor.fill(within(sheet, field('Code from claude.com')), 'nope');
            await visitor.click(within(sheet, button('Continue')));

            // Then - the engine's refusal, in its words
            await visitor.see(within(sheet, content('the code was refused', { exact: false })));
        });
        expect(result.html.text).not.toContain('nope');
    });

    test('an expired sign-in strikes its step, and offers a new code', async () => {
        // Given - claude's sign-in on linux-1 over sign-ins, its code never pasted
        const result = await component.render(signInsOver('sign-ins'), async (visitor) => {
            // When - Sign in of claude is clicked and the engine lets the code run out
            await visitor.click(within(region('Sign-ins'), button('Sign in claude')));
            const sheet = dialog(PASTE);

            // Then - the engine's expired word and line, and a way to start again
            await visitor.see(
                within(sheet, content('expired before Claude was approved', { exact: false })),
            );
            await visitor.see(within(sheet, button('New code')));
            await visitor.click(within(sheet, button('New code')));
            await visitor.see(within(sheet, button('Open claude.com')));
        });
        expect(result.errors.text).toBe('');
    });

    test('cancel closes the sheet and drops its watch', async () => {
        // Given - gh's sign-in on linux-1 over two-macs, its code shown
        const result = await component.render(signInsOver('two-macs'), async (visitor) => {
            await visitor.click(within(region('Sign-ins'), button('Sign in gh')));
            const sheet = dialog(DEVICE_CODE);
            await visitor.see(within(sheet, content('WDJB-MJHT')));

            // When - Cancel is clicked
            await visitor.click(within(sheet, button('Cancel')));

            // Then - the sheet is gone, and the row still offers its sign-in
            await visitor.gone(sheet);
            await visitor.see(within(region('Sign-ins'), button('Sign in gh')));
        });
        expect(result.errors.text).toBe('');
    });
});
