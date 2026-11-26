export type Pin = 'hidden' | 'pinned';

// value: the window width, in px, from which the column is pinned when no choice was made; the window opens 1440 wide.
export const PINNED_FROM = 1440;

// invariant: a choice made wins over the width and is kept; with none, the window's width alone decides.
export function pinnedOf(chosen: Pin | undefined, width: number): boolean {
    return chosen === undefined ? width >= PINNED_FROM : chosen === 'pinned';
}
