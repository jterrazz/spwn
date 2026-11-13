import { IDLE } from './gesture';
import type { Ask, GestureState } from './gesture';

export type Confirmation = {
    key: string;
    title: string;
    verb: string;
    destroys: boolean;
    confirm: (typed: string) => void;
    cancel: () => void;
};

export type Confirming = Confirmation & { ask: Ask };

type Entry = {
    state: GestureState<unknown>;
    listeners: Set<() => void>;
    subscribe: (listener: () => void) => () => void;
    grace: ReturnType<typeof setTimeout> | undefined;
};

export type GestureTiming = { grace: number };

const DEFAULTS: GestureTiming = { grace: 1500 };

function settled(state: GestureState<unknown>): boolean {
    return state.kind !== 'running' && state.kind !== 'asking';
}

// invariant: one state per gesture request compared by value (its call, input and target), so a row the engine's
// invariant: next answer leaves out for a moment finds its gesture as it was when it comes back. A settled state
// invariant: is forgotten a grace after its last subscriber leaves; a running one when it settles with none.
export class GestureStore {
    private readonly timing: GestureTiming;
    private readonly entries = new Map<string, Entry>();
    private readonly confirmations = new Map<string, Confirmation>();
    private readonly confirmingListeners = new Set<() => void>();
    private shown: Confirming | undefined;
    private shownFrom: Confirmation | undefined;

    constructor(timing: Partial<GestureTiming> = {}) {
        this.timing = { ...DEFAULTS, ...timing };
    }

    readonly state = (key: string): GestureState<unknown> => this.entries.get(key)?.state ?? IDLE;

    // value: the entry's own function once one is held, so React keeps its subscription; a render creates nothing.
    readonly subscriberOf = (key: string): ((listener: () => void) => () => void) =>
        this.entries.get(key)?.subscribe ?? ((listener) => this.entryOf(key).subscribe(listener));

    readonly set = (key: string, state: GestureState<unknown>): void => {
        const entry = this.entryOf(key);
        entry.state = state;
        for (const listener of entry.listeners) {
            listener();
        }
        this.forgetLater(key, entry);
        if (this.confirmations.has(key)) {
            if (settled(state)) {
                this.confirmations.delete(key);
            }
            this.showConfirming();
        }
    };

    readonly confirming = (): Confirming | undefined => this.shown;

    readonly subscribeConfirming = (listener: () => void): (() => void) => {
        this.confirmingListeners.add(listener);
        return () => {
            this.confirmingListeners.delete(listener);
        };
    };

    readonly askedBy = (confirmation: Confirmation): void => {
        this.confirmations.delete(confirmation.key);
        this.confirmations.set(confirmation.key, confirmation);
        this.showConfirming();
    };

    // invariant: one confirmation is shown at a time, the one of the gesture last started that asks; it lives here,
    // invariant: not in the button that started it, so a row rebuilt while it asks leaves it open.
    private showConfirming(): void {
        const asking = [...this.confirmations.values()].findLast(
            (one) => this.state(one.key).kind === 'asking',
        );
        const state = asking === undefined ? undefined : this.state(asking.key);
        const ask = state?.kind === 'asking' ? state.ask : undefined;
        if (asking === this.shownFrom && ask === this.shown?.ask) {
            return;
        }
        this.shownFrom = asking;
        this.shown = asking !== undefined && ask !== undefined ? { ...asking, ask } : undefined;
        for (const listener of this.confirmingListeners) {
            listener();
        }
    }

    private entryOf(key: string): Entry {
        const existing = this.entries.get(key);
        if (existing) {
            return existing;
        }
        const entry: Entry = {
            grace: undefined,
            listeners: new Set(),
            state: IDLE,
            subscribe: (listener) => {
                clearTimeout(entry.grace);
                entry.grace = undefined;
                entry.listeners.add(listener);
                return () => {
                    entry.listeners.delete(listener);
                    this.forgetLater(key, entry);
                };
            },
        };
        this.entries.set(key, entry);
        return entry;
    }

    private forgetLater(key: string, entry: Entry): void {
        if (entry.listeners.size > 0 || !settled(entry.state)) {
            return;
        }
        clearTimeout(entry.grace);
        entry.grace = setTimeout(() => {
            if (entry.listeners.size === 0 && this.entries.get(key) === entry) {
                this.entries.delete(key);
            }
        }, this.timing.grace);
    }
}
