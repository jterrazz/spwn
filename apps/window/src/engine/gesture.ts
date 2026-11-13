import type { Progress } from '@jterrazz/station-api';

import type { EngineError } from './engine-error';

export type AskForm = 'name' | 'yes';

export type Ask = { form: AskForm; name: string; refusal: EngineError | undefined };

export type GestureState<O> =
    | { kind: 'asking'; ask: Ask }
    | { kind: 'done'; out: O }
    | { kind: 'failed'; error: EngineError }
    | { kind: 'idle' }
    | { kind: 'running'; progress: readonly Progress[]; answer: string | undefined }
    | { kind: 'unknown'; error: EngineError };

export type GestureEvent<O> =
    | { kind: 'answered'; out: O }
    | { kind: 'cancel' }
    | { kind: 'confirm'; answer: string }
    | { kind: 'dismiss' }
    | { kind: 'lost'; error: EngineError }
    | { kind: 'progress'; progress: Progress }
    | { kind: 'refused'; error: EngineError }
    | { kind: 'start'; ask: Ask | undefined };

export const IDLE = { kind: 'idle' } as const;

const YES = 'yes';

export function answerOf(ask: Ask, typed: string): string | undefined {
    if (ask.form === 'yes') {
        return YES;
    }
    return typed === ask.name ? typed : undefined;
}

// invariant: the engine's own confirmation refusal says what it asks (`detail.level`: `yes-no` or `type-name`)
// invariant: and of what (`detail.name`, or its subject); apps/engine/core/confirm/confirm.go writes both on every one.
export function askOf(error: EngineError): Ask | undefined {
    const { wire } = error;
    if (wire?.type !== 'needs-confirmation') {
        return undefined;
    }
    const level = wire.detail?.level;
    const name = wire.detail?.name ?? wire.subject ?? '';
    if (level === 'type-name') {
        return { form: 'name', name, refusal: error };
    }
    return level === 'yes-no' ? { form: 'yes', name, refusal: error } : undefined;
}

function settled<O>(state: GestureState<O>, event: GestureEvent<O>): GestureState<O> {
    if (event.kind === 'start') {
        return event.ask
            ? { ask: event.ask, kind: 'asking' }
            : { answer: undefined, kind: 'running', progress: [] };
    }
    return event.kind === 'dismiss' ? IDLE : state;
}

function running<O>(
    state: Extract<GestureState<O>, { kind: 'running' }>,
    event: GestureEvent<O>,
): GestureState<O> {
    switch (event.kind) {
        case 'progress': {
            return { ...state, progress: [...state.progress, event.progress] };
        }
        case 'answered': {
            return { kind: 'done', out: event.out };
        }
        case 'lost': {
            return { error: event.error, kind: 'unknown' };
        }
        case 'refused': {
            const ask = askOf(event.error);
            return ask ? { ask, kind: 'asking' } : { error: event.error, kind: 'failed' };
        }
        default: {
            return state;
        }
    }
}

function asking<O>(
    state: Extract<GestureState<O>, { kind: 'asking' }>,
    event: GestureEvent<O>,
): GestureState<O> {
    if (event.kind === 'cancel') {
        return IDLE;
    }
    const answer = event.kind === 'confirm' ? answerOf(state.ask, event.answer) : undefined;
    if (answer !== undefined) {
        return { answer, kind: 'running', progress: [] };
    }
    return state;
}

export function step<O>(state: GestureState<O>, event: GestureEvent<O>): GestureState<O> {
    switch (state.kind) {
        case 'asking': {
            return asking(state, event);
        }
        case 'running': {
            return running(state, event);
        }
        default: {
            return settled(state, event);
        }
    }
}
