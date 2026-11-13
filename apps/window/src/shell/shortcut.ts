import { TABS } from './route';
import type { Tab } from './route';

export type Keystroke = { key: string; metaKey: boolean; ctrlKey: boolean; shiftKey: boolean };

export type Shortcut = 'back' | 'find' | 'forward' | 'gallery' | 'home' | Tab;

const COMMANDS: Readonly<Record<string, 'back' | 'find' | 'forward' | 'home'>> = {
    '0': 'home',
    '[': 'back',
    ']': 'forward',
    k: 'find',
};

const MOUSE_BUTTONS: Readonly<Record<number, 'back' | 'forward'>> = { 3: 'back', 4: 'forward' };

export function mouseShortcutOf(button: number): 'back' | 'forward' | undefined {
    return MOUSE_BUTTONS[button];
}

export function shortcutOf(stroke: Keystroke, withGallery: boolean): Shortcut | undefined {
    if (!stroke.metaKey && !stroke.ctrlKey) {
        return undefined;
    }
    const key = stroke.key.toLowerCase();
    if (withGallery && stroke.shiftKey && key === 'g') {
        return 'gallery';
    }
    if (stroke.shiftKey) {
        return undefined;
    }
    if (Object.hasOwn(COMMANDS, key)) {
        return COMMANDS[key];
    }
    const digit = Number(key);
    return /^[1-9]$/u.test(key) ? TABS[digit - 1] : undefined;
}
