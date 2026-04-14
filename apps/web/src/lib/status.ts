/**
 * Consistent status → color mapping used across the entire app.
 */

export type WorldStatus = 'creating' | 'error' | 'idle' | 'running' | 'stopped';

/** Tailwind color classes for status dot indicators */
export const STATUS_DOT: Record<string, string> = {
    running: 'bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)] animate-pulse',
    idle: 'bg-amber-400 shadow-[0_0_6px_rgba(251,191,36,0.5)]',
    error: 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.6)]',
    stopped: 'bg-zinc-500/40',
    creating: 'bg-blue-400 shadow-[0_0_6px_rgba(96,165,250,0.5)]',
};

/** Tailwind text color for status text */
export const STATUS_TEXT: Record<string, string> = {
    running: 'text-green-400',
    idle: 'text-amber-400',
    error: 'text-red-400',
    stopped: 'text-zinc-400',
    creating: 'text-blue-400',
};

/** Badge-style classes for status badges */
export const STATUS_BADGE: Record<string, string> = {
    running: 'bg-green-500/10 text-green-400 border-green-500/20',
    idle: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
    error: 'bg-red-500/10 text-red-400 border-red-500/20',
    stopped: 'bg-zinc-500/10 text-zinc-400 border-zinc-500/20',
    creating: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
};

/** Role badge colors */
export const ROLE_BADGE: Record<string, string> = {
    chief: 'bg-amber-500/10 text-amber-300 border-amber-500/20',
    manager: 'bg-purple-500/10 text-purple-300 border-purple-500/20',
    worker: 'bg-blue-500/10 text-blue-300 border-blue-500/20',
    npc: 'bg-zinc-500/10 text-zinc-400 border-zinc-500/20',
    default: 'bg-zinc-500/10 text-zinc-400 border-zinc-500/20',
};
