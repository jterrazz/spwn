import type { Space } from './spaces';

// invariant: the space whose Soon list is open, held apart from the frame: opening or closing it draws the bar's
// invariant: spaces alone, never the page; one list at a time, and pressing the open one again closes it.
export class Soon {
    private space: Space | undefined;
    private readonly listeners = new Set<() => void>();

    readonly subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };

    readonly shown = (): Space | undefined => this.space;

    readonly toggle = (space: Space): void => {
        this.set(this.space === space ? undefined : space);
    };

    readonly close = (): void => {
        this.set(undefined);
    };

    private set(space: Space | undefined): void {
        if (space === this.space) {
            return;
        }
        this.space = space;
        for (const listener of this.listeners) {
            listener();
        }
    }
}
