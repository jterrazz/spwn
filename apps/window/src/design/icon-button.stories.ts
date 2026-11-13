import type { IconButtonProps } from './icon-button';
import { nothing } from './story';
import type { Story } from './story';

export const ICON_BUTTON_STORIES: readonly Story<IconButtonProps>[] = [
    {
        name: 'plain',
        props: { icon: 'sidebar', label: 'Show or hide the sidebar', onClick: nothing },
    },
    {
        name: 'with its shortcut',
        props: { icon: 'back', label: 'Back', onClick: nothing, shortcut: '⌘[' },
    },
    { name: 'find', props: { icon: 'find', label: 'Find', onClick: nothing, shortcut: '⌘K' } },
    {
        name: 'disabled',
        props: { disabled: true, icon: 'forward', label: 'Forward', onClick: nothing },
    },
];
