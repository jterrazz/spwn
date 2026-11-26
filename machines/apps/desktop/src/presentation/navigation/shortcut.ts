import { TABS } from './route';
import type { Tab } from './route';
import { SPACES } from './spaces';

export type Keystroke = { key: string; metaKey: boolean; ctrlKey: boolean; shiftKey: boolean };

export type Shortcut =
    | 'back'
    | 'column'
    | 'find'
    | 'forward'
    | 'gallery'
    | 'home'
    | 'inbox'
    | 'settings'
    | 'space-agent'
    | 'space-machine'
    | 'space-os'
    | Tab;

const COMMANDS: Readonly<
    Record<string, 'back' | 'find' | 'forward' | 'home' | 'inbox' | 'settings'>
> = {
    ',': 'settings',
    '0': 'home',
    '[': 'back',
    ']': 'forward',
    i: 'inbox',
    k: 'find',
};

// value: ⌘ with a digit is a space, one per member of SPACES, in their order.
const SPACE_KEYS: Readonly<Record<string, 'space-agent' | 'space-machine' | 'space-os'>> =
    Object.fromEntries(SPACES.map((space, at) => [String(at + 1), `space-${space}` as const]));

const MOUSE_BUTTONS: Readonly<Record<number, 'back' | 'forward'>> = { 3: 'back', 4: 'forward' };

export function mouseShortcutOf(button: number): 'back' | 'forward' | undefined {
    return MOUSE_BUTTONS[button];
}

export function shortcutOf(stroke: Keystroke, withGallery: boolean): Shortcut | undefined {
    if (!stroke.metaKey && !stroke.ctrlKey) {
        return undefined;
    }
    const key = stroke.key.toLowerCase();
    if (stroke.shiftKey) {
        return withGallery && key === 'g' ? 'gallery' : undefined;
    }
    if (stroke.metaKey && stroke.ctrlKey && key === 's') {
        return 'column';
    }
    if (Object.hasOwn(COMMANDS, key)) {
        return COMMANDS[key];
    }
    if (stroke.metaKey) {
        return SPACE_KEYS[key];
    }
    return /^[1-9]$/u.test(key) ? TABS[Number(key) - 1] : undefined;
}
