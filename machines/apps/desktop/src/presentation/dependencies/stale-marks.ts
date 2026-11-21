import type { EngineError } from '../../domain/engine-error';

// value: the reads drawn now whose last refresh was refused, each held under its reader's own id, so the frame
// value: says once, quietly, that what it shows is older than it looks, and stops saying it when they refresh.
export class StaleMarks {
    readonly #held = new Map<string, EngineError>();
    readonly #listeners = new Set<() => void>();
    #shown: readonly EngineError[] = [];

    hold(reader: string, stale: EngineError | undefined): void {
        if (this.#held.get(reader) === stale) {
            return;
        }
        if (stale === undefined) {
            this.#held.delete(reader);
        } else {
            this.#held.set(reader, stale);
        }
        this.#shown = [...new Set(this.#held.values())];
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

    readonly shown = (): readonly EngineError[] => this.#shown;
}
