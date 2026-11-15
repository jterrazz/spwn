import type { CoreCallWireError } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { EngineError } from '../engine-error';
import { askOf, IDLE, step } from '../gesture';
import type { Ask, GestureEvent, GestureState } from '../gesture';

type State = GestureState<string>;

const ASK_YES: Ask = { form: 'yes', name: 'linux-1', refusal: undefined };
const ASK_NAME: Ask = { form: 'name', name: 'api', refusal: undefined };

function refusal(wire: CoreCallWireError): EngineError {
    return new EngineError(wire);
}

const TYPED_NEEDED = refusal({
    code: 'confirm.needed',
    detail: { action: 'stop', kind: 'server', level: 'type-name', name: 'infra', typed: 'yes' },
    message: 'stopping a server needs its name',
    subject: 'infra',
    type: 'needs-confirmation',
});

const STOPPED = refusal({
    hint: 'station engine start',
    message: 'the engine is stopped on This Mac',
});

function walk(events: readonly GestureEvent<string>[], from: State = IDLE): State {
    let state = from;
    for (const event of events) {
        state = step(state, event);
    }
    return state;
}

describe('gesture', () => {
    test('a call with no confirmation runs at once, then is done with its out', () => {
        // Given - an idle gesture
        // When - it starts with nothing to ask, hears a progress, then its out
        const running = walk([{ ask: undefined, kind: 'start' }]);
        const done = walk(
            [
                { kind: 'progress', progress: { task: 'Updating the mise apps' } },
                { kind: 'answered', out: 'ok' },
            ],
            running,
        );

        // Then - it ran with no answer, and settled on the out
        expect(running).toStrictEqual({ answer: undefined, kind: 'running', progress: [] });
        expect(done).toStrictEqual({ kind: 'done', out: 'ok' });
    });

    test('a call that declares a confirmation asks before it sends, and a yes runs it', () => {
        // Given - an idle gesture of a call with a confirmation
        // When - it starts, then the owner says yes
        const asking = walk([{ ask: ASK_YES, kind: 'start' }]);
        const running = step(asking, { answer: '', kind: 'confirm' });

        // Then - it asked first, then ran with the engine's yes
        expect(asking).toStrictEqual({ ask: ASK_YES, kind: 'asking' });
        expect(running).toStrictEqual({ answer: 'yes', kind: 'running', progress: [] });
    });

    test('a cancel at the confirm sends nothing and returns to idle', () => {
        // Given - a gesture asking
        const asking = walk([{ ask: ASK_YES, kind: 'start' }]);

        // When - the owner cancels
        const after = step(asking, { kind: 'cancel' });

        // Then - nothing runs
        expect(after).toStrictEqual(IDLE);
    });

    test('a name asked runs only on the name typed whole', () => {
        // Given - a gesture asking for the name api
        const asking = walk([{ ask: ASK_NAME, kind: 'start' }]);

        // When - a wrong name, then the right one, is typed
        const wrong = step(asking, { answer: 'ap', kind: 'confirm' });
        const right = step(wrong, { answer: 'api', kind: 'confirm' });

        // Then - the wrong one keeps it asking, the right one is the answer sent
        expect(wrong).toBe(asking);
        expect(right).toStrictEqual({ answer: 'api', kind: 'running', progress: [] });
    });

    test("the engine's confirmation refusal asks again in the form it names", () => {
        // Given - a gesture that ran on a yes
        const running = walk([
            { ask: ASK_YES, kind: 'start' },
            { answer: '', kind: 'confirm' },
        ]);

        // When - the engine refuses: this machine needs its name typed
        const after = step(running, { error: TYPED_NEEDED, kind: 'refused' });

        // Then - it asks for the name the engine names, with the refusal shown
        expect(after).toStrictEqual({
            ask: { form: 'name', name: 'infra', refusal: TYPED_NEEDED },
            kind: 'asking',
        });
    });

    test('an error after progress fails the gesture with the error', () => {
        // Given - a running gesture that heard a step fail
        const failedStep = { error: { message: 'brew exited 1' }, task: 'Updating the brew apps' };
        const running = walk([
            { ask: undefined, kind: 'start' },
            { kind: 'progress', progress: failedStep },
        ]);

        // When - the engine answers an error
        const after = step(running, { error: STOPPED, kind: 'refused' });

        // Then - the progress was kept as it came, and the gesture failed on the error
        expect(running).toStrictEqual({
            answer: undefined,
            kind: 'running',
            progress: [failedStep],
        });
        expect(after).toStrictEqual({ error: STOPPED, kind: 'failed' });
    });

    test('a settled gesture is dismissed or started again, and a running one ignores both', () => {
        // Given - a failed gesture and a running one
        const failed: State = { error: STOPPED, kind: 'failed' };
        const running: State = { answer: 'yes', kind: 'running', progress: [] };

        // When - each is dismissed, and started again
        // Then - a settled one goes idle or asks again; a running one stays running
        expect(step(failed, { kind: 'dismiss' })).toStrictEqual(IDLE);
        expect(step(failed, { ask: ASK_YES, kind: 'start' })).toStrictEqual({
            ask: ASK_YES,
            kind: 'asking',
        });
        expect(step(running, { kind: 'dismiss' })).toBe(running);
        expect(step(running, { ask: ASK_YES, kind: 'start' })).toBe(running);
    });

    test('only a needs-confirmation refusal with a level is a question', () => {
        // Given - the engine's refusals
        const yesNeeded = refusal({
            code: 'confirm.yes-needed',
            detail: { level: 'yes-no', name: 'linux-1' },
            message: 'refusing without --yes: linux-1',
            type: 'needs-confirmation',
        });

        // When - each is read as a question
        // Then - a level is a form; any other error is none
        expect(askOf(yesNeeded)).toStrictEqual({
            form: 'yes',
            name: 'linux-1',
            refusal: yesNeeded,
        });
        expect(askOf(TYPED_NEEDED)?.form).toBe('name');
        expect(askOf(STOPPED)).toBeUndefined();
        expect(askOf(new EngineError(undefined, 'answered'))).toBeUndefined();
    });

    test('a gesture whose answer is lost is unknown, never failed, and can be made again', () => {
        // Given - a running gesture whose connection dropped after it was sent
        const lost = new EngineError(undefined, 'the engine closed the connection', 'lost');
        const running = walk([{ ask: undefined, kind: 'start' }]);

        // When - the loss is heard, then it is dismissed, then started again
        const unknown = step(running, { error: lost, kind: 'lost' });
        const dismissed = step(unknown, { kind: 'dismiss' });
        const again = step(unknown, { ask: undefined, kind: 'start' });

        // Then - its outcome is unknown, carrying the reason, until dismissed or made again
        expect(unknown).toStrictEqual({ error: lost, kind: 'unknown' });
        expect(dismissed).toBe(IDLE);
        expect(again).toStrictEqual({ answer: undefined, kind: 'running', progress: [] });
    });
});
