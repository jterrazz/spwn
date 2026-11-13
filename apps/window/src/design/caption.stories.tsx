import type { ComponentProps } from 'react';

import type { Caption } from './caption';
import { DotStrip } from './dot-strip';
import type { Story } from './story';
import { Value } from './value';

export const CAPTION_STORIES: readonly Story<ComponentProps<typeof Caption>>[] = [
    { name: 'a word', props: { children: 'Managers' } },
    {
        name: 'a layer: its value, its count, its dots',
        props: {
            children: (
                <>
                    <Value identity="teal" look="raised">
                        jterrazz-base
                    </Value>
                    <span>3 apps</span>
                    <DotStrip identity="teal" names={['jq', 'git', 'node']} />
                </>
            ),
        },
    },
    {
        name: 'long text',
        props: {
            children:
                'A caption heads a group of rows, and wraps when the group it names is long enough to need two lines',
        },
    },
];
