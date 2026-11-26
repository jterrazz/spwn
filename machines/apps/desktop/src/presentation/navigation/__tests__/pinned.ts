import type { Memory } from '../route-memory';

// value: the storage of a window whose column was pinned, as the launch after the pin reads it.
export const PINNED: Memory = {
    getItem: (key) => (key === 'station.window.column' ? 'pinned' : null),
    setItem: () => undefined,
};

// value: the storage of a window whose column was hidden, as the launch after the choice reads it.
export const HIDDEN: Memory = {
    getItem: (key) => (key === 'station.window.column' ? 'hidden' : null),
    setItem: () => undefined,
};
