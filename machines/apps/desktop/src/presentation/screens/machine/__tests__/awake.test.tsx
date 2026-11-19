import { button, component, content, dialog, file, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf(
    'stories',
    'two-macs',
    'kept-four-hours',
    'awake-stopped',
    'engine-stopped',
);

function overviewOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Overview' });
}

function pressedTerms(): (null | string)[] {
    return [...document.querySelectorAll('[role="dialog"] [aria-pressed="true"]')].map(
        (term) => term.textContent,
    );
}

describe('awake', () => {
    test("a Mac that sleeps as usual says so first, then offers the engine's terms in one row, none pressed, and no Stop", async () => {
        // Given - This Mac's Overview over two-macs, whose awake.read says Sleeps normally., three choices, none picked, no Session
        let pressed: (null | string)[] = [''];
        const result = await component.render(
            overviewOver('two-macs', 'macbook-pro'),
            async (visitor) => {
                // When - Awake… is pressed
                await visitor.click(button('Keep macbook-pro awake'));

                // Then - the sheet names the Mac, says its state, and offers each choice the engine labelled
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.see(within(sheet, content('Sleeps normally.')));
                await visitor.see(within(sheet, button('1 h')));
                await visitor.see(within(sheet, button('4 h')));
                await visitor.see(within(sheet, button('Until 18:00')));
                await visitor.gone(within(sheet, button('Stop keeping awake')));
                pressed = pressedTerms();
            },
        );
        expect(pressed).toStrictEqual([]);
        await expect(result.tree).toMatch(file('./__expected__/two-macs-awake-sheet.aria.yaml'));
    });

    test('a running keep says until when, presses the term the engine picked, and puts Stop beside Close', async () => {
        // Given - This Mac's Overview over stories, whose awake.read answers a Session until 18:00, lid closed, Until 18:00 picked
        let pressed: (null | string)[] = [];
        const result = await component.render(
            overviewOver('stories', 'macbook-pro'),
            async (visitor) => {
                // When - Awake… is pressed
                await visitor.click(button('Keep macbook-pro awake'));

                // Then - the state line says the engine's sentence, and Stop keeping awake is offered
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.see(within(sheet, content('Awake until 18:00, lid closed.')));
                await visitor.see(within(sheet, button('Stop keeping awake')));
                pressed = pressedTerms();
            },
        );
        expect(pressed).toStrictEqual(['Until 18:00']);
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/stories-awake-sheet.aria.yaml'));
    });

    test('a keep asked for four hours presses 4 h, the one the engine picked', async () => {
        // Given - This Mac's Overview over kept-four-hours, whose Session was asked for 4h, so its 4 h choice is picked
        let pressed: (null | string)[] = [];
        const result = await component.render(
            overviewOver('kept-four-hours', 'macbook-pro'),
            async (visitor) => {
                // When - Awake… is pressed
                await visitor.click(button('Keep macbook-pro awake'));

                // Then - the sheet says until when, and offers Stop
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.see(within(sheet, content('Awake until 14:00.')));
                await visitor.see(within(sheet, button('Stop keeping awake')));
                pressed = pressedTerms();
            },
        );
        expect(pressed).toStrictEqual(['4 h']);
        expect(result.errors.text).toBe('');
    });

    test('a keep a guard stopped says why, presses no term, and offers no Stop', async () => {
        // Given - This Mac's Overview over awake-stopped, whose engine ended the last keep two minutes ago on battery, no Session
        let pressed: (null | string)[] = [''];
        const result = await component.render(
            overviewOver('awake-stopped', 'macbook-pro'),
            async (visitor) => {
                // When - Awake… is pressed
                await visitor.click(button('Keep macbook-pro awake'));

                // Then - the engine's warning sentence leads, and the three terms are offered again
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.see(within(sheet, content('Awake stopped, battery 18 %.')));
                await visitor.see(within(sheet, button('Until 18:00')));
                await visitor.gone(within(sheet, button('Stop keeping awake')));
                pressed = pressedTerms();
            },
        );
        expect(pressed).toStrictEqual([]);
        await expect(result.tree).toMatch(
            file('./__expected__/awake-stopped-awake-sheet.aria.yaml'),
        );
    });

    test('a Mac that never sleeps says so with its line, presses no term, and still offers the terms', async () => {
        // Given - the mini's Overview over two-macs, whose awake.read says Never sleeps. with the policy's line, none picked
        let pressed: (null | string)[] = [''];
        const result = await component.render(
            overviewOver('two-macs', 'mac-mini'),
            async (visitor) => {
                // When - the mini's Awake… is pressed
                await visitor.click(button('Keep mac-mini awake'));

                // Then - the sentence, then its line, then the terms
                const sheet = dialog('Keep mac-mini awake');
                await visitor.see(within(sheet, content('Never sleeps.')));
                await visitor.see(within(sheet, content('the always-on Mac policy')));
                await visitor.see(within(sheet, button('1 h')));
                pressed = pressedTerms();
            },
        );
        expect(pressed).toStrictEqual([]);
        await expect(result.tree).toMatch(
            file('./__expected__/two-macs-mini-awake-sheet.aria.yaml'),
        );
    });

    test('a Mac whose engine is stopped says it sleeps as usual, and why', async () => {
        // Given - This Mac's Overview over engine-stopped, whose awake.read says Sleeps normally. with the engine's line
        const result = await component.render(
            overviewOver('engine-stopped', 'macbook-pro'),
            async (visitor) => {
                // When - Awake… is pressed
                await visitor.click(button('Keep macbook-pro awake'));

                // Then - the sentence and its line stand above the terms
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.see(within(sheet, content('Sleeps normally.')));
                await visitor.see(within(sheet, content('the engine is stopped')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a term pressed keeps the Mac awake with the choice the engine sent, and the sheet closes', async () => {
        // Given - This Mac's Overview over two-macs, whose awake.keep is recorded for the 4 h choice's For
        const result = await component.render(
            overviewOver('two-macs', 'macbook-pro'),
            async (visitor) => {
                // When - Awake…, then 4 h, are pressed
                await visitor.click(button('Keep macbook-pro awake'));
                await visitor.click(within(dialog('Keep macbook-pro awake'), button('4 h')));

                // Then - the keep is answered and the sheet is gone
                await visitor.gone(dialog('Keep macbook-pro awake'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a far Mac opens its own sheet, of the choices its engine answered', async () => {
        // Given - the mini's Overview over stories, whose engine answers three choices
        const far = await component.render(overviewOver('stories', 'mac-mini'), async (visitor) => {
            // When - the mini's Awake… is pressed
            await visitor.click(button('Keep mac-mini awake'));

            // Then - its sheet offers the choices the mini's engine answered
            await visitor.see(within(dialog('Keep mac-mini awake'), content('1 h')));
        });
        expect(far.errors.text).toBe('');
    });
});
