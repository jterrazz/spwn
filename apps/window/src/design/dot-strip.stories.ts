import type { ComponentProps } from 'react';

import type { DotStrip } from './dot-strip';
import type { Story } from './story';

export const DOT_STRIP_STORIES: readonly Story<ComponentProps<typeof DotStrip>>[] = [
    { name: 'a few', props: { identity: 'teal', names: ['jq', 'git', 'node'] } },
    {
        name: 'many',
        props: {
            identity: 'pink',
            names: Array.from({ length: 40 }, (_, index) => `app-${String(index)}`),
        },
    },
    { name: 'the system identity', props: { identity: 'system', names: ['macos', 'xcode'] } },
    { name: 'empty', props: { identity: 'blue', names: [] } },
];
