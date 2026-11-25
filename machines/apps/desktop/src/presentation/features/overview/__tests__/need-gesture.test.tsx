import { button, component, content, dialog, field, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import type { Recording } from '../../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';

const recordingFor = await recordingsOf('engine-stopped', 'machinefile-differs', 'two-macs');

// value: each frame of a watch held this long, so a step is on screen before the next one replaces it.
const PACE = { gestureFrame: 600 };

const needs = region('Needs you');

const waiting = region('Upkeep');

const doneHere = region('Done here');

const RESTART = {
    Args: { Name: 'infra' },
    Broken: true,
    Call: 'machines.restart',
    Command: '',
    Confirm: '',
    Machine: 'infra',
    Mark: 'down',
    Noun: 'Overview',
    On: 'mac-mini',
    Path: '',
    Section: 'overview',
    Severity: 'broken',
    Subject: 'infra',
    Text: 'infra check fails',
    Upkeep: 'needs-you',
    Verb: 'restart',
};

const TRY_AGAIN = {
    Args: { VM: 'api' },
    Broken: true,
    Call: 'machines.start',
    Command: 'station machines start api',
    Confirm: '',
    Machine: 'api',
    Mark: 'down',
    Noun: 'Overview',
    On: 'macbook-pro',
    Path: '',
    Section: 'overview',
    Severity: 'broken',
    Subject: 'api',
    Text: 'api was not created: it failed at boot',
    Upkeep: 'needs-you',
    Verb: 'try-again',
};

function recordOf(value: unknown): Record<string, unknown> | undefined {
    return typeof value === 'object' && value !== null
        ? Object.fromEntries(Object.entries(value))
        : undefined;
}

function needed(answer: unknown, list: readonly object[]): unknown {
    const reply = recordOf(answer);
    const out = recordOf(reply?.out);
    const summary = recordOf(out?.Summary);
    return reply && out && summary?.Pending === false
        ? { ...reply, out: { ...out, Summary: { ...summary, Needs: list } } }
        : answer;
}

function withNeeds(recording: Recording, list: readonly object[]): Recording {
    return {
        ...recording,
        exchanges: recording.exchanges.map((exchange) =>
            exchange.request.watch === 'overview.read' && exchange.request.target === undefined
                ? { ...exchange, answers: exchange.answers.map((one) => needed(one, list)) }
                : exchange,
        ),
    };
}

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
});
