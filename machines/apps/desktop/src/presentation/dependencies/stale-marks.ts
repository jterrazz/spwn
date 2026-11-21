import type { EngineError } from '../../domain/engine-error';

// value: a read drawn whose last refresh was refused: what the engine said, and when the answer still shown was read.
export type Stale = { error: EngineError; at: Date };

function same(before: Stale | undefined, next: Stale | undefined): boolean {
    return before?.error === next?.error && before?.at.getTime() === next?.at.getTime();
}

// value: one mark per refusal, as of the oldest answer it keeps on screen.
function shownOf(held: Iterable<Stale>): readonly Stale[] {
    const byError = new Map<EngineError, Stale>();
    for (const one of held) {
        const seen = byError.get(one.error);
        if (seen === undefined || one.at.getTime() < seen.at.getTime()) {
            byError.set(one.error, one);
        }
    }
    return [...byError.values()];
}

// value: the reads drawn now whose last refresh was refused, each held under its reader's own id, so the frame
// value: says once, quietly, that what it shows is older than it looks, and stops saying it when they refresh.
export class StaleMarks {
    readonly #held = new Map<string, Stale>();
    readonly #listeners = new Set<() => void>();
    #shown: readonly Stale[] = [];

    hold(reader: string, stale: Stale | undefined): void {
        if (same(this.#held.get(reader), stale)) {
            return;
        }
        if (stale === undefined) {
            this.#held.delete(reader);
        } else {
            this.#held.set(reader, stale);
        }
        this.#shown = shownOf(this.#held.values());
        for (const listener of this.#listeners) {
            listener();
        }
    }

    readonly subscribe = (listener: () => void): (() => void) => {
        this.#listeners.add(listener);
        return () => {
            this.#listeners.delete(listener);
        };
    };

    readonly shown = (): readonly Stale[] => this.#shown;
}
