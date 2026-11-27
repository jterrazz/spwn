import { button, component, content, dialog, field, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';
import {
    DELETE,
    FIX_LINE,
    PANE,
    PATH_ONLY,
    RESTART,
    TRY_AGAIN,
    UPGRADE,
    withNeeds,
} from './need-gesture.fixtures';

const recordingFor = await recordingsOf(
    'create-failed',
    'engine-stopped',
    'machinefile-differs',
    'two-macs',
);

// value: each frame of a watch held this long, so a step is on screen before the next one replaces it.
const PACE = { gestureFrame: 600 };

const needs = region('Needs you');

const waiting = region('Upkeep');

const doneHere = region('Done here');

function brightNames(): string[] {
    return [...document.querySelectorAll('button')]
        .filter((one) => one.className.includes('primary'))
        .map((one) => one.getAttribute('aria-label') ?? '');
}

describe('NeedLine', () => {
    test('a need whose fix is a call runs it, and leaves the list once the engine no longer reports it', async () => {
        // Given - the Inbox over two-macs, where one mise update waits
        const result = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(waiting, content('1 mise update waiting')));

                // When - its fix, apps.update, is clicked
                await visitor.click(within(waiting, button('Update mise')));

                // Then - no dialog, the need read again is gone, and its outcome is held in Done here
                await visitor.see(within(doneHere, content('1 mise update waiting')));
                await visitor.see(within(doneHere, content('done')));
                await visitor.gone(within(waiting, button('Update mise')));
                await visitor.gone(dialog());
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a background task the engine names is removed once its removal is confirmed', async () => {
        // Given - the Inbox over two-macs, where a background task is failing and presses
        const failing = 'com.example.brain.x-snapshot is failing, last exit 2';
        const asked = 'Remove com.example.brain.x-snapshot?';
        const result = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(needs, content(failing)));

                // When - its fix is clicked, then confirmed in the dialog confirms.json asks
                await visitor.click(within(needs, button('Remove com.example.brain.x-snapshot')));
                await visitor.see(dialog(asked));
                await visitor.click(within(dialog(asked), button('Remove')));

                // Then - overview.read, read again, no longer lists it, and Done here holds its outcome
                await visitor.gone(dialog(asked));
                await visitor.see(within(doneHere, content(failing)));
                await visitor.gone(within(needs, button('Remove com.example.brain.x-snapshot')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a cancelled removal sends nothing and leaves the need listed', async () => {
        // Given - the Inbox over machinefile-differs, its failing background task listed
        const asked = 'Remove com.example.brain.x-snapshot?';
        const result = await component.render(
            pageOver(recordingFor('machinefile-differs'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.click(within(needs, button('Remove com.example.brain.x-snapshot')));

                // When - its dialog is cancelled
                await visitor.click(within(dialog(asked), button('Cancel')));

                // Then - the need and its button stay, and nothing is held as done
                await visitor.gone(dialog(asked));
                await visitor.see(within(needs, button('Remove com.example.brain.x-snapshot')));
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
                await visitor.see(within(waiting, content('~/.tart is misplaced')));

                // When - its fix, disk.move, is clicked
                await visitor.click(within(waiting, button('Move ~/.tart')));

                // Then - no dialog, and the need read again is held as done
                await visitor.see(within(doneHere, content('~/.tart is misplaced')));
                await visitor.gone(within(waiting, button('Move ~/.tart')));
                await visitor.gone(dialog());
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a server's failing check is restarted from its need, through the dialog its page asks, and it is the one bright button", async () => {
        // Given - the Inbox over two-macs, its one need infra's failing check
        const recording = withNeeds(recordingFor('two-macs'), [RESTART]);
        let bright: string[] = [];
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(needs, content('infra check fails', { exact: false })));
                bright = brightNames();

                // When - the Restart of its need is clicked
                await visitor.click(within(needs, button('Restart infra')));

                // Then - the dialog of its page asks yes or no, no name typed
                const asked = dialog('Restart infra?');
                await visitor.see(within(asked, button('Restart')));
                await visitor.gone(within(asked, field('Type infra to confirm')));
            },
        );

        // Then - and only this button was drawn primary
        expect(bright).toStrictEqual(['Restart infra']);
        expect(result.errors.text).toBe('');
    });

    test('a need in the section that names its state draws its mark alone, and a need in another state keeps its word', async () => {
        // Given - Home over create-failed: linux-2 not created, and a background task that needs you
        const said: { words: string[]; marks: string[] } = { marks: [], words: [] };
        const result = await component.render(
            pageOver(recordingFor('create-failed'), { page: 'home' }),
            async (visitor) => {
                // When - its Needs you section is drawn
                await visitor.see(within(needs, button('Try again linux-2')));
                const section = document.querySelector('section[aria-label="Needs you"]');
                const states = [...(section?.querySelectorAll('[data-mark]') ?? [])];
                said.words = states.map((one) => one.textContent).filter((word) => word !== '');
                said.marks = states.flatMap((one) =>
                    one.getAttribute('role') === 'img' ? [one.getAttribute('aria-label') ?? ''] : [],
                );
            },
        );

        // Then - the need that needs you draws its diamond named, the one not created keeps its word
        expect(said).toStrictEqual({ marks: ['Needs you'], words: ['Not created'] });
        expect(result.errors.text).toBe('');
    });

    test('a VM whose create failed is started again from its need, worded as the engine words it', async () => {
        // Given - the Inbox over two-macs, its one need api, whose create failed
        const recording = withNeeds(recordingFor('two-macs'), [TRY_AGAIN]);
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                // Then - its button wears Try again, never Start
                await visitor.see(within(needs, button('Try again api')));
                await visitor.gone(within(needs, button('Start api')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a broken sign-in's need opens the same sign-in sheet the Sign-ins tab opens, aimed at the need's machine", async () => {
        // Given - the Inbox over two-macs, where This Mac's npm sign-in is broken
        const result = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'inbox' }, NO_OPENER, PACE),
            async (visitor) => {
                // When - its Fix is clicked
                await visitor.click(within(needs, button('Fix npm')));

                // Then - the engine's step for npm on This Mac is on a sheet, which goes once the engine signs in
                const sheet = dialog('Sign in to npm on macbook-pro');
                await visitor.see(sheet);
                await visitor.gone(sheet);
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a need whose fix is a path alone draws the path under its sentence and no verb', async () => {
        // Given - the Inbox over two-macs, its one need a firewall that is off, fixed by a path alone
        const recording = withNeeds(recordingFor('two-macs'), [PATH_ONLY]);
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                // Then - the path is read as a quiet line, and nothing stands for a verb
                await visitor.see(within(needs, content('System Settings > Network > Firewall')));
                await visitor.gone(within(needs, content('Fix')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a need whose fix is a pane of This Mac draws the button that opens it, and no command', async () => {
        // Given - the Inbox over two-macs, its one need This Mac's firewall, with the pane the engine sent
        const recording = withNeeds(recordingFor('two-macs'), [PANE]);
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                // Then - Open Firewall settings stands for the verb, and nothing copies a command
                await visitor.see(within(needs, button('Open Firewall settings')));
                await visitor.gone(within(needs, button('Copy the command')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a pane of another Mac is never drawn as a button here', async () => {
        // Given - the Inbox over two-macs, a firewall need of mac-mini that sends its pane
        const recording = withNeeds(recordingFor('two-macs'), [
            { ...PANE, Machine: 'mac-mini', On: 'mac-mini' },
        ]);
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                // Then - the need is listed with no button to open a pane on this Mac
                await visitor.see(within(needs, content('Firewall: off')));
                await visitor.gone(within(needs, button('Open Firewall settings')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a line the owner runs is a button that fills it into Terminal, never shown', async () => {
        // Given - the Inbox over two-macs, its one need a root task whose removal the owner runs
        const recording = withNeeds(recordingFor('two-macs'), [FIX_LINE]);
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                // Then - Run in Terminal stands for the verb, the line itself is not on the page
                await visitor.see(within(needs, button('Run in Terminal')));
                await visitor.gone(within(needs, content('sudo launchctl', { exact: false })));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a VM that is gone is deleted from its need, through the dialog that asks its name typed, and no command is offered', async () => {
        // Given - the Inbox over two-macs, its one need a VM whose create failed and left nothing to start
        const recording = withNeeds(recordingFor('two-macs'), [DELETE]);
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(needs, button('Delete api')));

                // When - its Delete is clicked
                await visitor.click(within(needs, button('Delete api')));

                // Then - the dialog of its page asks the name typed, and the need never offered to copy a command
                await visitor.see(within(dialog('Delete api?'), field('Type api to confirm')));
                await visitor.gone(content('Copy the command'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a night's failed upgrade is run again from its need, and no command is offered", async () => {
        // Given - the Inbox over two-macs, its one need the night's upgrade that failed
        const recording = withNeeds(recordingFor('two-macs'), [UPGRADE]);
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                // Then - its button wears Upgrade, and nothing offers the command
                await visitor.see(within(needs, button('Upgrade macbook-pro')));
                await visitor.gone(content('Copy the command'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
