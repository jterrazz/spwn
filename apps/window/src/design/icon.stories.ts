import type { ComponentProps } from 'react';

import type { Icon, IconName } from './icon';
import type { Story } from './story';

const NAMES: readonly IconName[] = [
    'sidebar',
    'find',
    'back',
    'forward',
    'home',
    'inbox',
    'changes',
    'file',
    'box',
    'disc',
    'mac',
    'workspace',
    'server',
    'apps',
    'skills',
    'settings',
    'check',
    'close',
];

export const ICON_STORIES: readonly Story<ComponentProps<typeof Icon>>[] = NAMES.map((name) => ({
    name,
    props: { name },
}));
