import { button, component, content, dialog, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('engine-stopped', 'machinefile-differs', 'two-macs');

const needs = region('Needs you');

const doneHere = region('Done here');

describe('NeedLine', () => {
    test('a need whose fix is a call runs it, and leaves the list once the engine no longer reports it', async () => {
        // Given - the Inbox over two-macs, where one mise update waits
        const result = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(needs, content('1 mise update waiting')));

                // When - its fix, apps.update, is clicked
                await visitor.click(within(needs, button('Update mise')));

                // Then - no dialog, the need read again is gone, and its outcome is held in Done here
                await visitor.see(within(doneHere, content('1 mise update waiting')));
                await visitor.see(within(doneHere, content('Done')));
                await visitor.gone(within(needs, button('Update mise')));
                await visitor.gone(dialog());
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a background task the engine names is removed once its removal is confirmed', async () => {
        // Given - Home over two-macs, where a background task is failing
        const failing = 'com.jterrazz.brain.x-snapshot is failing, last exit 2';
        const asked = 'Remove com.jterrazz.brain.x-snapshot?';
        const result = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'home' }),
            async (visitor) => {
                await visitor.see(within(needs, content(failing)));

                // When - its fix is clicked, then confirmed in the dialog calls.json asks
                await visitor.click(within(needs, button('Remove com.jterrazz.brain.x-snapshot')));
                await visitor.see(dialog(asked));
                await visitor.click(within(dialog(asked), button('Remove')));

                // Then - overview.read, read again, no longer lists it, and Done here holds its outcome
                await visitor.gone(dialog(asked));
                await visitor.see(within(doneHere, content(failing)));
                await visitor.gone(within(needs, button('Remove com.jterrazz.brain.x-snapshot')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a cancelled removal sends nothing and leaves the need listed', async () => {
        // Given - the Inbox over machinefile-differs, its failing background task listed
        const asked = 'Remove com.jterrazz.brain.x-snapshot?';
        const result = await component.render(
            pageOver(recordingFor('machinefile-differs'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.click(within(needs, button('Remove com.jterrazz.brain.x-snapshot')));

                // When - its dialog is cancelled
                await visitor.click(within(dialog(asked), button('Cancel')));

                // Then - the need and its button stay, and nothing is held as done
                await visitor.gone(dialog(asked));
                await visitor.see(within(needs, button('Remove com.jterrazz.brain.x-snapshot')));
                await visitor.gone(doneHere);
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a stopped engine is started by its need, with no dialog, and its need goes', async () => {
        // Given - the Inbox over engine-stopped, whose one need is the stopped engine
        const result = await component.render(
            pageOver(recordingFor('engine-stopped'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(needs, content('Engine stopped')));

                // When - engine.start is clicked
                await visitor.click(within(needs, button('Start the engine', { exact: false })));

                // Then - no confirmation is asked, and the need the overview no longer lists is held as done
                await visitor.gone(dialog());
                await visitor.see(within(doneHere, content('Engine stopped', { exact: false })));
                await visitor.gone(within(needs, button('Start the engine', { exact: false })));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a misplaced folder is moved from its need, which leaves the list', async () => {
        // Given - the Inbox over two-macs, where ~/.tart is misplaced
        const result = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(needs, content('~/.tart is misplaced')));

                // When - its fix, disk.move, is clicked
                await visitor.click(within(needs, button('Move ~/.tart')));

                // Then - no dialog, and the need read again is held as done
                await visitor.see(within(doneHere, content('~/.tart is misplaced')));
                await visitor.gone(within(needs, button('Move ~/.tart')));
                await visitor.gone(dialog());
            },
        );
        expect(result.errors.text).toBe('');
    });
});
