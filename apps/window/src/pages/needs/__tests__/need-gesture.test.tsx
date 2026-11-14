import { button, component, content, dialog, field, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { Recording } from '../../../engine/replay';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('engine-stopped', 'machinefile-differs', 'two-macs');

const needs = region('Needs you');

const waiting = region('Can wait');

const doneHere = region('Done here');

const RESTART = {
    Args: { Name: 'infra' },
    Broken: true,
    Call: 'machines.restart',
    Command: '',
    Machine: 'infra',
    Path: '',
    Section: 'overview',
    Severity: 'broken',
    Subject: 'infra',
    Text: 'infra check fails',
    Upkeep: 'needs-you',
    Verb: 'restart',
};

const DELETE = {
    Args: { VM: 'linux-1' },
    Broken: false,
    Call: 'machines.delete',
    Command: '',
    Machine: 'linux-1',
    Path: '',
    Section: 'overview',
    Severity: 'needs-a-look',
    Subject: 'linux-1',
    Text: 'linux-1 has been idle for 5 h and holds 10 GB',
    Upkeep: 'can-wait',
    Verb: 'delete',
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
                await visitor.see(within(doneHere, content('Done')));
                await visitor.gone(within(waiting, button('Update mise')));
                await visitor.gone(dialog());
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a background task the engine names is removed once its removal is confirmed', async () => {
        // Given - the Inbox over two-macs, where a background task is failing
        const failing = 'com.jterrazz.brain.x-snapshot is failing, last exit 2';
        const asked = 'Remove com.jterrazz.brain.x-snapshot?';
        const result = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(waiting, content(failing)));

                // When - its fix is clicked, then confirmed in the dialog calls.json asks
                await visitor.click(
                    within(waiting, button('Remove com.jterrazz.brain.x-snapshot')),
                );
                await visitor.see(dialog(asked));
                await visitor.click(within(dialog(asked), button('Remove')));

                // Then - overview.read, read again, no longer lists it, and Done here holds its outcome
                await visitor.gone(dialog(asked));
                await visitor.see(within(doneHere, content(failing)));
                await visitor.gone(within(waiting, button('Remove com.jterrazz.brain.x-snapshot')));
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
                await visitor.click(
                    within(waiting, button('Remove com.jterrazz.brain.x-snapshot')),
                );

                // When - its dialog is cancelled
                await visitor.click(within(dialog(asked), button('Cancel')));

                // Then - the need and its button stay, and nothing is held as done
                await visitor.gone(dialog(asked));
                await visitor.see(within(waiting, button('Remove com.jterrazz.brain.x-snapshot')));
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
        // Given - the Inbox over two-macs, where infra's check fails and linux-1 has sat idle
        const recording = withNeeds(recordingFor('two-macs'), [RESTART, DELETE]);
        let bright: string[] = [];
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(needs, content('infra check fails', { exact: false })));
                bright = brightNames();

                // When - the Restart of its need is clicked
                await visitor.click(within(needs, button('Restart infra')));

                // Then - the dialog of its page asks the name typed
                const asked = dialog('Restart infra?');
                await visitor.see(within(asked, field('Type infra to confirm')));
            },
        );

        // Then - and only this button was drawn primary
        expect(bright).toStrictEqual(['Restart infra']);
        expect(result.errors.text).toBe('');
    });

    test('an idle workspace is deleted from its need only by its name typed, and the dialog can be left', async () => {
        // Given - the same Inbox, linux-1's delete waiting among the needs that can wait
        const recording = withNeeds(recordingFor('two-macs'), [RESTART, DELETE]);
        const result = await component.render(
            pageOver(recording, { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(
                    within(waiting, content('linux-1 has been idle for 5 h', { exact: false })),
                );

                // When - its Delete is clicked, then the dialog is cancelled
                await visitor.click(within(waiting, button('Delete linux-1')));
                const asked = dialog('Delete linux-1?');
                await visitor.see(within(asked, field('Type linux-1 to confirm')));
                await visitor.click(within(asked, button('Cancel')));

                // Then - nothing is sent, and the need stays
                await visitor.gone(dialog());
                await visitor.see(within(waiting, button('Delete linux-1')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
