// invariant: the codes `src-tauri/src/menu_bar.rs` refuses a read or a switch with, and no other.
export const MENU_BAR_CODES = ['menubar.not-read', 'menubar.not-switched'] as const;

export type MenuBarCode = (typeof MENU_BAR_CODES)[number];

export function isMenuBarCode(code: unknown): code is MenuBarCode {
    return MENU_BAR_CODES.some((known) => known === code);
}

// value: the nested menu bar's login item as macOS holds it: shown at every login, hidden, registered and waiting
// value: for the owner's approval in System Settings, or a window with no switch (a debug build, off macOS).
export type MenuBarRead =
    | { kind: 'approval' }
    | { kind: 'hidden' }
    | { kind: 'shown' }
    | { kind: 'unsupported' };

export type MenuBarRefused = { kind: 'refused'; code: MenuBarCode; message: string };
