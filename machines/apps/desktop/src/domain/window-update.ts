// invariant: the codes `src-tauri/src/updates.rs` refuses a check or an install with, and no other.
export const WINDOW_UPDATE_CODES = [
    'window.update-busy',
    'window.update-gone',
    'window.update-install',
    'window.update-unpublished',
    'window.update-unreachable',
    'window.update-unsigned',
] as const;

export type WindowUpdateCode = (typeof WINDOW_UPDATE_CODES)[number];

export function isWindowUpdateCode(code: unknown): code is WindowUpdateCode {
    return WINDOW_UPDATE_CODES.some((known) => known === code);
}

export type WindowChecked =
    | { kind: 'current'; current: string }
    | { kind: 'out'; current: string; version: string; date?: string }
    | { kind: 'unsupported' };

export type WindowRefused = { kind: 'refused'; code: WindowUpdateCode; message: string };

export type WindowProgress =
    | { kind: 'downloaded'; downloaded: number; total?: number }
    | { kind: 'installed' };

export type WindowInstalled = { kind: 'installed' };
