import type { IconButtonProps } from './icon-button';
import { nothing } from './story';
import type { Story } from './story';

export const ICON_BUTTON_STORIES: readonly Story<IconButtonProps>[] = [
    {
        name: 'plain',
        props: { icon: 'gear', label: 'Settings', onClick: nothing },
    },
    {
        name: 'with its shortcut',
        props: { icon: 'back', label: 'Back', onClick: nothing, shortcut: '⌘[' },
    },
    {
        name: 'pressed',
        props: {
            icon: 'sidebar',
            label: 'Pin the column',
            onClick: nothing,
            pressed: true,
            shortcut: '⌃⌘S',
        },
    },
    {
        name: 'disabled',
        props: { disabled: true, icon: 'forward', label: 'Forward', onClick: nothing },
    },
];
