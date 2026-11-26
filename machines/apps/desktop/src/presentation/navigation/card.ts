import type { Route } from './route';

// invariant: whether the card is open, held apart from the frame: opening, closing or ⌘L draws the path's first
// invariant: segment and the card alone, never the page. It keeps the last machine route the window showed, where
// invariant: the card's Fleet goes.
export class Card {
    private open = false;
    private machine: Route | undefined;
    private readonly listeners = new Set<() => void>();

    readonly subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };

    readonly shown = (): boolean => this.open;

    readonly show = (): void => {
        this.set(true);
    };

    readonly toggle = (): void => {
        this.set(!this.open);
    };

    readonly close = (): void => {
        this.set(false);
    };

    readonly seen = (route: Route): void => {
        if (route.page === 'machine') {
            this.machine = route;
        }
    };

    readonly lastMachine = (): Route | undefined => this.machine;

    private set(open: boolean): void {
        if (open === this.open) {
            return;
        }
        this.open = open;
        for (const listener of this.listeners) {
            listener();
        }
    }
}
