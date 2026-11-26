import { recalledPin, rememberPin } from './column-memory';
import { pinnedOf } from './pin';
import type { Pin } from './pin';
import type { Memory } from './route-memory';

export type Placed = { pinned: boolean; peeking: boolean };

// value: how long a peeked column stays once the pointer has left it, in ms.
export const LEAVES_AFTER = 250;

// invariant: the column's state, held apart from the frame: pinning it or peeking it tells only who reads it, so the
// invariant: page beneath is not drawn again. Every gesture is one function for the window's life.
export class ColumnState {
    private readonly memory: Memory | undefined;
    private chosen: Pin | undefined;
    private width: number;
    private shown: Placed;
    private leaving: ReturnType<typeof setTimeout> | undefined;
    private readonly listeners = new Set<() => void>();

    constructor(memory: Memory | undefined, width: number) {
        this.memory = memory;
        this.chosen = recalledPin(memory);
        this.width = width;
        this.shown = { peeking: false, pinned: pinnedOf(this.chosen, width) };
    }

    readonly subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };

    readonly read = (): Placed => this.shown;

    readonly toggle = (): void => {
        this.choose(this.shown.pinned ? 'hidden' : 'pinned');
    };

    readonly pin = (): void => {
        this.choose('pinned');
    };

    readonly peek = (): void => {
        this.stay();
        if (!this.shown.pinned) {
            this.show({ peeking: true, pinned: false });
        }
    };

    readonly stay = (): void => {
        clearTimeout(this.leaving);
        this.leaving = undefined;
    };

    readonly leave = (): void => {
        this.stay();
        this.leaving = setTimeout(this.close, LEAVES_AFTER);
    };

    readonly close = (): void => {
        this.stay();
        this.show({ peeking: false, pinned: this.shown.pinned });
    };

    readonly resized = (width: number): void => {
        this.width = width;
        this.show({ peeking: this.shown.peeking, pinned: pinnedOf(this.chosen, width) });
    };

    private choose(pin: Pin): void {
        this.stay();
        this.chosen = pin;
        rememberPin(this.memory, pin);
        this.show({ peeking: false, pinned: pinnedOf(pin, this.width) });
    }

    private show(next: Placed): void {
        const { pinned } = next;
        const peeking = next.peeking && !pinned;
        if (pinned === this.shown.pinned && peeking === this.shown.peeking) {
            return;
        }
        this.shown = { peeking, pinned };
        for (const listener of this.listeners) {
            listener();
        }
    }
}
