import type { EngineSliceOut } from '@jterrazz/station-api';

import { EngineError } from '../../domain/engine-error';
import { viewNamed } from '../../domain/views';
import type { View } from '../../domain/views';
import type { Asked, Reading } from './entry';
import { replaceEqualDeep } from './equal';

function staleOf(slice: EngineSliceOut): EngineError | undefined {
    return slice.state === 'stale' ? new EngineError(undefined, slice.why ?? 'stale') : undefined;
}

function atOf(slice: EngineSliceOut, held: Reading<unknown> | undefined): Date {
    if (slice.read_at !== undefined) {
        return new Date(slice.read_at);
    }
    return held?.kind === 'read' ? held.at : new Date(0);
}

// value: the why of a view the engine could not read and holds no reading of: its call answers the refusal, so the
// value: window asks it; `reading` is a view still being read, which draws its skeleton.
export function refusedOf(slice: EngineSliceOut, held: Reading<unknown>): string | undefined {
    const refused =
        slice.state === 'unread' &&
        slice.value === undefined &&
        slice.why !== undefined &&
        slice.why !== 'reading' &&
        held.kind !== 'read';
    return refused ? slice.why : undefined;
}

function sameMark(held: Reading<unknown>, stale: EngineError | undefined): boolean {
    return held.kind === 'read' && held.stale?.message === stale?.message;
}

// invariant: a slice replaces the reading it moves and keeps the parts that did not: an equal value is the one shown,
// invariant: a slice sent without its value keeps the value held, and a slice not read yet draws nothing new.
export function readingOf(before: Reading<unknown>, slice: EngineSliceOut): Reading<unknown> {
    const held = before.kind === 'read' ? before : undefined;
    if (slice.value === undefined && held === undefined) {
        return before;
    }
    const out = replaceEqualDeep(held?.out, slice.value ?? held?.out);
    const stale = staleOf(slice);
    if (held && out === held.out && sameMark(held, stale)) {
        return held;
    }
    const at = atOf(slice, held);
    return stale
        ? { at, done: true, kind: 'read', out, stale }
        : { at, done: true, kind: 'read', out };
}

// value: the view a read is, by the engine's naming (`domain/views.ts`); undefined is a read asked by its call.
export function viewOf(asked: Asked): undefined | View {
    return viewNamed(asked.name, asked.input, asked.target);
}
